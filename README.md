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

## CI/CD

このプロジェクトはGitHub Actionsを使用した継続的インテグレーションを実装しています。

### 自動テスト
- **トリガー**: 
  - 任意のブランチへのプッシュ
  - 任意のブランチへのプルリクエスト
- **実行環境**: Ubuntu latest（Docker事前インストール済み）
- **Goバージョン**: 1.21
- **実行内容**:
  1. コードのチェックアウト
  2. Go環境のセットアップ
  3. Dockerの可用性確認
  4. 依存関係のダウンロード
  5. テストの実行（詳細出力付き）

### Docker環境
- GitHub ActionsのUbuntuランナーにはDockerが事前インストールされています
- テスト実行時に必要なDockerイメージは自動的にプルされます：
  - `bfren/ftps:latest`（FTPSテスト用）
  - `garethflowers/ftp-server:latest`（通常FTPテスト用）

ワークフローファイル: `.github/workflows/test.yml`
  