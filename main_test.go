package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"
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
	mountPath   string
	username    string
	password    string
	pasvMinPort int
	pasvMaxPort int
}

func TestFTPServer(t *testing.T) {
	// Table-driven test cases
	testCases := []testCase{
		{
			name:       "FTP (plain)",
			repository: "fauria/vsftpd",
			tag:        "latest",
			useTLS:     false,
			envVars: []string{
				"FTP_USER=testuser",
				"FTP_PASS=testpass",
				"PASV_ADDRESS=127.0.0.1",
				"PASV_MIN_PORT=21100",
				"PASV_MAX_PORT=21110",
			},
			dialOptions: []ftp.DialOption{
				ftp.DialWithTimeout(5 * time.Second),
				ftp.DialWithDisabledEPSV(true),
			},
			mountPath:   "/home/vsftpd/testuser",
			username:    "testuser",
			password:    "testpass",
			pasvMinPort: 21100,
			pasvMaxPort: 21110,
		},
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
			mountPath:   "/files",
			username:    "ftps",
			password:    "pass",
			pasvMinPort: 60000,
			pasvMaxPort: 60010,
		},
		{
			name:       "FTP (garethflowers)",
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
			mountPath:   "/home/ftps",
			username:    "ftps",
			password:    "pass",
			pasvMinPort: 40000,
			pasvMaxPort: 40010,
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

			absTestdataPath, err := filepath.Abs("testdata")
			if err != nil {
				t.Fatalf("Could not get absolute path for testdata: %v", err)
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
				Mounts: []string{
					fmt.Sprintf("%s:%s", absTestdataPath, tc.mountPath),
				},
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

			// Give the container time to fully initialize
			time.Sleep(1 * time.Second)

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

			// List files and verify expected files exist
			entries, err := ftpConn.List("")
			if err != nil {
				t.Fatalf("Could not list files: %v", err)
			}

			fileNames := make(map[string]bool)
			for _, entry := range entries {
				fileNames[entry.Name] = true
			}

			expectedFiles := []string{"file1.txt", "file2.txt"}
			for _, expectedFile := range expectedFiles {
				if !fileNames[expectedFile] {
					t.Errorf("Expected file %s not found", expectedFile)
				}
			}
		})
	}
}
