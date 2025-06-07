Go言語による次のテストを行うプログラムです。

- main_test.go
  - github.com/ory/dockertest/v3を用い、dockerイメージfauria/vsftpdで、testdataをマウントしてFTPサーバを立てる
  - FTP（平文）とFTPS（TLS暗号化）の両方をテーブル駆動型でテスト
  - そのFTPサーバに接続し、ファイルリストを得る
  - file1.txt, file2.txt が得られたら成功

## テスト内容

1. **FTP (plain)** - 通常の非暗号化FTP接続（fauria/vsftpd使用）
2. **FTPS (with TLS)** - TLS暗号化されたFTP接続（bfren/ftps使用）
   - 証明書検証はスキップ（InsecureSkipVerify: true）
   - 注: bfren/ftpsはexplicit FTPS (AUTH TLS)を使用しています

## 実行方法

```bash
go test -v
```
  