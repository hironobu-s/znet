# Z.com cloud serucity group manager

[Z.com cloud](https://cloud.z.com/)のセキュリティグループを管理するためのツールです。

Z.com cloudには仮想サーバーが繋がっている仮想スイッチ側にパケットフィルタが備わっています。サーバー上のOSのファイアーウォール機能(iptables, firewalld)に影響されずに利用できるので便利ですが、[API](https://cloud.z.com/jp/guide/identity-get_version_list/)でしか操作することができません。このツールはそれらをコマンドラインで操作できるようにするものです。

[OpenStackのコマンドラインツール](http://docs.openstack.org/cli-reference/)でも同じことができますが、機能を絞っている分```znet```の方が簡単に使えると思います。

**Note:** このツールは[hironobu-s/conoha-net](https://github.com/hironobu-s/conoha-net)をforkして作成されています。

## できること

* 通信方向(Ingress / Egress)
* プロトコルの種類(TCP / UDP / ICMP)
* プロトコルのバージョン(IPv4 / IPv6)
* 接続元IPアドレス,もしくはIPレンジ

以上の組み合わせによるパケットフィルタリング。

## インストール

任意のディレクトリに実行ファイルをダウンロードして実行して下さい。

**Mac OSX**

```shell
curl -sL https://github.com/hironobu-s/znet/releases/download/current/znet-osx.amd64.gz | zcat > znet && chmod +x ./znet
```


**Linux(amd64)**

```bash
curl -sL https://github.com/hironobu-s/znet/releases/download/current/znet-linux.amd64.gz | zcat > znet && chmod +x ./znet
```

**Windows(amd64)**

[ZIP file](https://github.com/hironobu-s/znet/releases/download/current/znet.amd64.zip)

## デフォルトルールについて

Z.com cloudにはデフォルトで下記のセキュリティグループが用意されていて変更/削除不可です。仮想サーバーへのアタッチ/デタッチは自由にできます。また**default**はアタッチしないと全ての通信が通らなくなるので、事実上必須となります。

### Z.com cloud
* default
* l-default
* n-default
* zjps-ipv4-*

znetのセキュリティグループを一覧表示するコマンドlist-groupは、デフォルトで**これらを表示しません**。--allオプションを明示的に指定する必要があります。

## 使い方

例として、mygroupと言う名前のセキュリティグループを作成して、133.130.0.0/16からTCP 22番ポート宛の通信のみを許可するルールを作成してみます。

### 1. 認証

znetを実行するには、APIの認証情報を環境変数にセットする必要があります。

認証情報は「APIユーザ名」「APIパスワード」「テナント名 or テナントID」です。これらの情報は[Z.com cloudのコントロールパネル](https://cp-jp.cloud.z.com/EP/API/)にあります。

以下はbashの例です。

```shell
export OS_USERNAME=[username]
export OS_PASSWORD=[password]
export OS_TENANT_NAME=[tenant name]
export OS_AUTH_URL=[identity endpoint]
export OS_REGION_NAME=[region]
```

参考: https://wiki.openstack.org/wiki/OpenStackClient/Authentication


### 2. セキュリティグループを作成する

create-groupで**my-group**と言う名前のセキュリティグループを作成します。

```
znet create-group my-group
```

list-groupを実行すると、今作ったセキュリティグループが表示されます。

```
# znet list-group
UUID                                     SecurityGroup     Direction     EtherType     Proto     IP Range     Port
05bb817c-5179-4156-99ec-f088ff5c5d8e     my-group          egress        IPv6          ALL                    ALL
5ecc4a23-0b92-4394-bca6-2466f08ef45e     my-group          egress        IPv4          ALL                    ALL
```


### 2. ルールを作成する

セキュリティグループにルールを追加することで、フィルタリングの挙動を設定します。これはcreate-ruleで行います。オプションは下記です。

```
OPTIONS:
   -d value, --direction value         (Required) The direction in which the rule applied. Must be either "ingress" or "egress" (default: "ingress")
   -e value, --ether-type value        (Required) Type of IP version. Must be either "Ipv4" or "Ipv6". (default: "IPv4")
   -p value, --port-range value        The source port or port range. For example "80", "80-8080".
   -P value, --protocol value          The IP protocol. Valid value are "tcp", "udp", "icmp" or "all". (default: "all")
   -g value, --remote-group-id value   The remote group ID to be associated with this rule.
   -i value, --remote-ip-prefix value  The IP prefix to be associated with this rule.
```

たとえば、133.130.0.0/16のIPレンジからのTCP 22番ポートへのインバウンド通信(ingress)を許可する場合は以下のように設定します。(-dオプションと-eオプションはデフォルト値があるので省略可能です)

```
znet create-rule -d ingress -e IPv4 -p 22 -P tcp -i 133.130.0.0/16 my-group
```

再度list-groupを実行すると、ルールが追加されていることが確認できます。

```shell
UUID                                     SecurityGroup     Direction     EtherType     Proto     IP Range           Port
05bb817c-5179-4156-99ec-f088ff5c5d8e     my-group          egress        IPv6          ALL                          ALL
5ecc4a23-0b92-4394-bca6-2466f08ef45e     my-group          egress        IPv4          ALL                          ALL
83e287b1-1bcd-425c-b162-8b2d5e008ddf     my-group          ingress       IPv4          tcp       133.130.0.0/16     22 - 22
```

### 3. 仮想サーバーにアタッチする

作成したセキュリティグループを一つ、もしくは複数の仮想サーバーにアタッチすることでフィルタリングが有効になります。これにはattachを使います。一つ目の引数にセキュリティグループをアタッチするIPアドレスを指定します。二つ目の引数は作成したセキュリティグループ名です。

```shell
znet attach [IP Address] my-group
```

その後listを実行すると、セキュリティグループがアタッチされたことを確認できます。

```
# znet list
NameTag          IPv4               IPv6      SecGroup
znet-test        163.44.***.***               default, my-group
```

## コマンド一覧

-hオプションでヘルプが表示されます。

```shell
NAME:
Z.com cloud security group manager

USAGE:
commands [global options] command [command options] [arguments...]

VERSION:
0.1

COMMANDS:
list          list all VPS
attach        attach a security group to VPS
detach        dettach a security group from VPS
list-group    list security groups and rules
create-group  create a security group
delete-group  delete a security group
create-rule   create a security group rule
delete-rule   delete a security group rule

GLOBAL OPTIONS:
--debug, -d    print debug informations.
--output value, -o value  specify output type. must be either "text" or "json". (default: "text")
--help, -h     show help
--version, -v  print the version
```

## サポート

このツールはオフィシャルなツールではありません。Z.com cloudのサポートへのお問い合わせはご遠慮ください。

## ライセンス

MIT
