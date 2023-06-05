# SRUEUI

[『SHOWROOM イベント 獲得ポイント一覧』(1) 関連ソースの公開について](https://zenn.dev/chouette2100/books/d8c28f8ff426b7/viewer/4fccae)

----------------------

これは[『SHOWROOM イベント 獲得ポイント一覧』(1) 関連ソースの公開について](https://zenn.dev/chouette2100/books/d8c28f8ff426b7/viewer/4fccae)から始まる一連の記事の一つです。

[Github - SRCGI](https://github.com/Chouette2100/SRCGI)（[『SHOWROOM イベント 獲得ポイント一覧』(3) 獲得ポイントの推移の表示とデータ取得対象イベント・配信者の設定](https://zenn.dev/chouette2100/books/d8c28f8ff426b7/viewer/56ec9b)）でイベントを登録したときイベントに参加しているルームの情報が保存されます。

このときデータ取得の対象となるルームの順位の範囲（例えば１〜１０）を指定するのですが、順位はイベントの進行にともなって変動するので、データ取得の対象としていなかったルームが指定順位範囲にはいってくる場合があります。
このモジュールは定期的に順位をチェックしてあらたに指定順位範囲になったルームをデータ取得の対象とするためのものです。

現時点ではcronで15分おきに起動されることを前提として作っています。デーモンにした方がいいようにも思えますが、cronだとモジュール差し替えが簡単なので今はこうしています。

cronでの起動の例

毎時10分から15分おきに起動しています。モジュールの起動は15分おきですが順位のチェックはイベント開始直後、イベント終了直前などの期間に応じて頻度がかわります。

```
$ crontab -e
.
.
10-59/15 * * * * ~chouette/MyProject/Showroom/SRUEUI/SRUEUI.sh
. 
.
```


---

上で実行しているシェルは以下のようなものです。

```
$ cd ~/MyProject/Showroom/SRUEUI.sh
$ cat SRUEUI.sh
#! /bin/sh
cd ~/MyProject/Showroom/SRUEUI
export DBNAME=xxxxxxxxx
export DBUSER=xxxxxxxxx
export DBPW=xxxxxxxx
./SRUEUI 1>>Error.log 2>>&1
```

データベースのログインパスワードはServerConfig.ymlの方に直接書いてもいいのですが、ここでは環境変数を介して与える方法を使っています。これはServerConfig.ymlが公開するパッケージに含まれているからというのが理由です。

---

ロードモジュールの作成と設置

現在 Linux Mint 21.1 Vera base: Ubuntu 22.04 jammy 、 go version go1.20.4 linux/amd64　で作成したロードモジュールを VPS（Ubuntu 20.04.4 LTS focal）に持っていって動かしているのですが、この場合次のような手順になります。

まず[Github - SRUEUI](https://github.com/Chouette2100/SRUEUI)から入手したソースを~/go/src/SRUEUI 以下におきます。

以下

```
$ cd ~/go/src/SRUEUI
$ go mod init
$ go mod tidy
$ CGO_ENABLED=0 go build SRUEUI.go
$ sftp -oServerAliveInterval=60 -i ~/.ssh/id_ed25519 -P nnnn xxxxxxxxnnn.nnn.nnn.nnn
sftp> cd ~/MyProject/Showroom/SRUEUI
sftp> put SRUEUI
sftp> put ServerConfig.yml
```

みたいな感じで進めます。

なお CGO_ENABLED=0 は最近VPSにもっていったときライブラリーのエラー（/lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.32' not found）が出るようになったので入れています、

VPSの方が
```
ldd (Ubuntu GLIBC 2.31-0ubuntu9.9) 2.31
```
でローカルの方が

```
ldd (Ubuntu GLIBC 2.35-0ubuntu3.1) 2.35
```
なので GLIBC_2.32 以上が必要ということでしょうか。

Goの場合ロードモジュールは必要なライブラリーをぜんぶ持ってるはず、なぜ？、ということでざっとググったところ、こういうエラーが起きるのはnet/http などのパッケージを使った場合内部的にはCライブラリを使う方法と純粋なGoですませる方法があり、何もしないと（CGO_ENABLED=0 を指定しないと）前者の方法になりCライブラリーがダイナミックリンクされてしまう、というのが原因のようです。このあたりの事情は正直よくわかってません。すみません。

なおVSCodeを使っている場合、tasks.jsonは次のような書き方でいいようです。ふつうと違うのはoptionsの部分です。

```
        {
            "label": "go build",
            "type": "shell",
            "options": {
                "env": {
                  "CGO_ENABLED": "0"
                }
            },
            "command": "go",
            "args": [
                "build",
                "-v",
                "./..." //  ここはこれでいいはずですが、どういうわけか私の環境では "SRUEUI.go"と書く必要がありました。どこか勘違いしてるような気がしますが...
            ],
            "problemMatcher": [],
            "group": {
                "kind": "build",
                "isDefault": true
            }
        },
```
