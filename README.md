Go言語による次のテストを行うプログラムです。

- main_test.go
  - github.com/ory/dockertest/v3を用い、dockerイメージwildscamp/vsftpdで、testdataをマウントしてFTPサーバを立てる
  - そのFTPサーバに接続し、ファイルリストを得る
  - file1.txt, file2.txt が得られたら成功
  