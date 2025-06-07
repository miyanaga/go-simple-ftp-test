package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type testCase struct {
	name        string
	repository  string
	tag         string
	useTLS      bool
	tlsConfig   *tls.Config
	envVars     []string
	dialOptions []ftp.DialOption
	username    string
	password    string
	pasvMinPort int
	pasvMaxPort int
}

func TestFTPServer(t *testing.T) {
	// Table-driven test cases
	testCases := []testCase{
		{
			name:       "FTPS (with TLS)",
			repository: "bfren/ftps",
			tag:        "latest",
			useTLS:     true,
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip TLS verification as requested
				MinVersion:         tls.VersionTLS10,
				MaxVersion:         tls.VersionTLS13,
			},
			envVars: []string{
				"BF_FTPS_EXTERNAL_IP=127.0.0.1",
				"BF_FTPS_VSFTPD_USER=ftps",
				"BF_FTPS_VSFTPD_PASS=pass",
				"BF_FTPS_VSFTPD_UID=1000",
				"BF_FTPS_VSFTPD_MIN_PORT=60000",
				"BF_FTPS_VSFTPD_MAX_PORT=60010",
			},
			dialOptions: []ftp.DialOption{
				ftp.DialWithTimeout(5 * time.Second),
				ftp.DialWithDisabledEPSV(true),
			},
			username:    "ftps",
			password:    "pass",
			pasvMinPort: 60000,
			pasvMaxPort: 60010,
		},
		{
			name:       "FTP (plain)",
			repository: "garethflowers/ftp-server",
			tag:        "latest",
			useTLS:     false,
			envVars: []string{
				"FTP_USER=ftps",
				"FTP_PASS=pass",
				"UID=1000",
				"GID=1000",
				"PUBLIC_IP=127.0.0.1",
			},
			dialOptions: []ftp.DialOption{
				ftp.DialWithTimeout(5 * time.Second),
				ftp.DialWithDisabledEPSV(true),
			},
			username:    "ftps",
			password:    "pass",
			pasvMinPort: 40000,
			pasvMaxPort: 40009,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := dockertest.NewPool("")
			if err != nil {
				t.Fatalf("Could not connect to docker: %v", err)
			}

			err = pool.Client.Ping()
			if err != nil {
				t.Fatalf("Could not connect to docker daemon: %v", err)
			}

			// Create port bindings
			portBindings := make(map[docker.Port][]docker.PortBinding)
			exposedPorts := []string{"21/tcp"}

			// Bind FTP control port
			portBindings["21/tcp"] = []docker.PortBinding{
				{HostIP: "0.0.0.0", HostPort: ""}, // Let Docker assign a free port
			}

			// For passive mode, expose and bind the port range
			if tc.pasvMinPort > 0 && tc.pasvMaxPort > 0 {
				for port := tc.pasvMinPort; port <= tc.pasvMaxPort; port++ {
					portStr := fmt.Sprintf("%d/tcp", port)
					exposedPorts = append(exposedPorts, portStr)
					portBindings[docker.Port(portStr)] = []docker.PortBinding{
						{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port)},
					}
				}
			}

			// Create container with appropriate configuration
			runOpts := &dockertest.RunOptions{
				Repository:   tc.repository,
				Tag:          tc.tag,
				Env:          tc.envVars,
				ExposedPorts: exposedPorts,
			}

			resource, err := pool.RunWithOptions(runOpts, func(config *docker.HostConfig) {
				config.AutoRemove = true
				config.RestartPolicy = docker.RestartPolicy{
					Name: "no",
				}
				config.PortBindings = portBindings
			})
			if err != nil {
				t.Fatalf("Could not start resource: %v", err)
			}

			defer func() {
				if err := pool.Purge(resource); err != nil {
					log.Printf("Could not purge resource: %v", err)
				}
			}()

			ftpPort := resource.GetPort("21/tcp")
			t.Logf("FTP server %s started on port %s", tc.name, ftpPort)

			// Give the container time to fully initialize
			time.Sleep(3 * time.Second)

			// Create FTP connection
			var ftpConn *ftp.ServerConn
			if err := pool.Retry(func() error {
				var err error

				// Configure dial options
				dialOpts := tc.dialOptions
				if tc.useTLS {
					dialOpts = append(dialOpts, ftp.DialWithExplicitTLS(tc.tlsConfig))
				}

				ftpConn, err = ftp.Dial(fmt.Sprintf("localhost:%s", ftpPort), dialOpts...)
				if err != nil {
					return err
				}

				err = ftpConn.Login(tc.username, tc.password)
				if err != nil {
					ftpConn.Quit()
					return err
				}

				return nil
			}); err != nil {
				t.Fatalf("Could not connect to %s server: %v", tc.name, err)
			}

			defer ftpConn.Quit()
			t.Logf("Successfully connected to %s server", tc.name)

			// Step 0: Check if SIZE and MDTM commands are supported
			t.Logf("Step 0: Checking SIZE and MDTM support for %s", tc.name)
			testContent := "Test file for commands"
			err = ftpConn.Stor("test_cmd.txt", strings.NewReader(testContent))
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Check SIZE support
			_, err = ftpConn.FileSize("test_cmd.txt")
			if err != nil {
				t.Errorf("SIZE command is NOT supported: %v", err)
			}

			// Check MDTM support
			_, err = ftpConn.GetTime("test_cmd.txt")
			if err != nil {
				t.Errorf("MDTM command is NOT supported: %v", err)
			}

			// Clean up
			ftpConn.Delete("test_cmd.txt")

			// Step 1: Create directory and files
			t.Logf("Step 1: Creating directory and files for %s", tc.name)
			err = ftpConn.MakeDir("dir1")
			if err != nil {
				t.Fatalf("Failed to create directory dir1: %v", err)
			}

			err = ftpConn.ChangeDir("dir1")
			if err != nil {
				t.Fatalf("Failed to change to directory dir1: %v", err)
			}

			err = ftpConn.Stor("file1.txt", strings.NewReader("This is file1 in dir1"))
			if err != nil {
				t.Fatalf("Failed to create dir1/file1.txt: %v", err)
			}

			err = ftpConn.ChangeDir("/")
			if err != nil {
				t.Fatalf("Failed to change back to root directory: %v", err)
			}

			err = ftpConn.Stor("file2.txt", strings.NewReader("This is file2 in root"))
			if err != nil {
				t.Fatalf("Failed to create file2.txt: %v", err)
			}

			err = ftpConn.Stor("file3.txt", strings.NewReader("This is file3 in root"))
			if err != nil {
				t.Fatalf("Failed to create file3.txt: %v", err)
			}

			// Step 2: List directories (root and dir1)
			t.Logf("Step 2: Listing directories for %s", tc.name)
			entries, err := ftpConn.List("")
			if err != nil {
				t.Fatalf("Failed to list root directory: %v", err)
			}

			// Check expected entries in root
			rootExpected := map[string]bool{
				"dir1":      false,
				"file2.txt": false,
				"file3.txt": false,
			}

			for _, entry := range entries {
				if _, exists := rootExpected[entry.Name]; exists {
					rootExpected[entry.Name] = true
				}
			}

			// Verify all expected entries were found
			for name, found := range rootExpected {
				if !found {
					t.Errorf("Expected entry '%s' not found in root directory", name)
				}
			}

			// List dir1 directory
			entries, err = ftpConn.List("dir1")
			if err != nil {
				t.Fatalf("Failed to list dir1 directory: %v", err)
			}

			// Check expected entries in dir1
			dir1Expected := map[string]bool{
				"file1.txt": false,
			}

			for _, entry := range entries {
				if _, exists := dir1Expected[entry.Name]; exists {
					dir1Expected[entry.Name] = true
				}
			}

			// Verify all expected entries were found
			for name, found := range dir1Expected {
				if !found {
					t.Errorf("Expected entry '%s' not found in dir1 directory", name)
				}
			}

			// Step 3: Check file size with SIZE command
			t.Logf("Step 3: Checking file size for %s", tc.name)
			expectedContent := "This is file2 in root"
			fileSize, err := ftpConn.FileSize("file2.txt")
			if err != nil {
				t.Fatalf("Failed to get size of file2.txt: %v", err)
			}

			if int(fileSize) != len(expectedContent) {
				t.Errorf("file2.txt size mismatch: got %d, expected %d", fileSize, len(expectedContent))
			}

			// Step 4: Read file content
			t.Logf("Step 4: Reading file content for %s", tc.name)
			response, err := ftpConn.Retr("dir1/file1.txt")
			if err != nil {
				t.Fatalf("Failed to retrieve dir1/file1.txt: %v", err)
			}
			defer response.Close()

			content, err := io.ReadAll(response)
			if err != nil {
				t.Fatalf("Failed to read dir1/file1.txt content: %v", err)
			}

			expectedFile1Content := "This is file1 in dir1"
			if string(content) != expectedFile1Content {
				t.Errorf("dir1/file1.txt content mismatch: got %q, expected %q", string(content), expectedFile1Content)
			}
		})
	}
}
