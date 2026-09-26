# SBAT6: Wi-Fi STA を足場にした有線 bridge/VLAN 解体

この文書は、所有または管理許可を得た SoftBank Air Terminal 6（SBAT6）で、
独立した Wi-Fi STA 管理経路を先に確立し、Linux 側の有線 bridge と VLAN
サブインターフェースを安全に外すための実測ベースの手順である。

公開用に整理した文書であり、実機の SSID、PSK、IP/MAC アドレス、SSH 鍵、
WebUI 認証情報、Cookie、RCE 実装、バックアップは含めない。

## 到達構成

```text
管理ホスト
  ├─ Wi-Fi LAN ── SBAT6 STA (apcli*) ── root SSH
  └─ 有線LAN  ── SBAT6 eth0           ── root SSH
```

最終構成では、Linux から見える有線データパスは `eth0` だけである。

| 項目 | 最終状態 |
| --- | --- |
| Linux bridge | `br-lan` は存在しない |
| Linux VLAN device | `eth0.1`、`eth0.2` は存在しない |
| 有線管理IP | `eth0` にのみ設定 |
| Wi-Fi 管理IP | 5 GHz STA が DHCP で取得 |
| 物理 switch | CPU 側を VLAN 1 の untagged member とする |
| root SSH | Wi-Fi と有線の両方から再起動後に確認 |

物理 switch の内部には forwarding domain として VLAN 1 が残る。これは
ハードウェア switch の通常動作であり、Linux が 802.1Q タグ付き
サブインターフェースを使用することとは別である。CPU 側が untagged なら、
Linux は通常の単一 NIC `eth0` として扱える。

## 安全条件

以下をすべて満たすまで、有線構成を変更しない。

1. 全 eMMC / partition / boot area の検証済みバックアップがある。
2. UART 救出経路または同等の復旧経路がある。
3. Wi-Fi STA 経由の root SSH が、通常再起動後にも復帰することを確認済み。
4. 有線と無線の管理経路を同時に失っても復旧できる。
5. RPMB、OTP/eFuse、secure-boot key、factory/NV/raw partition には書き込まない。

設定変更前には、少なくとも以下を `/data` のような永続領域に保存する。

```sh
ip -br addr
ip -d link
ip route
bridge link
brctl show 2>/dev/null || true
switch vlan dump egtag
uci show network
uci show dropbear
```

## Wi-Fi STA を先に恒久化する

### STA と parent radio のチャネル制約

この系列の無線ドライバでは、5 GHz の STA VIF と親 radio が同じチャネルを
共有する。親 radio が自動チャネル選択や DFS で移動すると STA が切断される。

したがって STA 接続先のチャネルを read-only で確認し、親 radio もその
非 DFS チャネルと帯域幅へ固定する。機種・driver によりインターフェース名は
異なるため、以下は概念例である。

```sh
# 例: 親 radio を起動し、接続先と同じ channel/bandwidth にする
ip link set <parent-radio> up
mwctl phy <phy> set channel num=<channel> bw=<bandwidth>

# 例: STA を起動後、vendor の supplicant 設定生成機能を使う
ip link set <sta-vif> up
wpa_cli -p /var/run/wpa_supplicant -i global interface_add \
  <sta-vif> <generated-conf>
wpa_cli -i <sta-vif> reconnect
udhcpc -i <sta-vif> -b -p /var/run/<sta-vif>.pid
```

PSK や SSID をシェル履歴・スクリプト・Git に直接書かない。既存の
vendor 設定ストアまたは運用時にのみ読み込む安全な秘密管理を使う。

### 管理ホスト側の経路

有線管理サブネットが Wi-Fi 側の大きいサブネットと重なる場合、Linux は
より長い有線 prefix を優先して、STA 宛のパケットを誤って有線へ送ることがある。
管理ホストでは、STA のホストアドレスだけを Wi-Fi IF へ送る `/32` 経路を
追加する。

```sh
# 文書用アドレス。実環境のSTAアドレス・Wi-Fi IF・sourceへ置換する。
ip route replace 192.0.2.13/32 dev <wifi-if> src <wifi-source-ip>
```

管理ホストに egress filter がある場合も、例外は対象 STA 一台、管理用 TCP
ポート、および実行ユーザーへ限定する。広い RFC1918 宛 allow や firewall の
無効化はしない。

DHCP 更新などで host route が消える環境では、`network-online.target` 後に
route を再適用する oneshot service と、定期的な systemd timer を組み合わせる。
timer は同じ限定 route と限定 firewall rule だけを再確認・再適用すること。

## Linux bridge の段階的解体

### 実行時テスト

Wi-Fi SSH を接続したまま、まず Linux bridge だけを外す。ここでは物理
switch の VLAN table を変更しない。

```sh
# 変更前の状態を永続領域へ保存してから行う。
ip addr flush dev br-lan
ip link set eth0.1 nomaster
ip link set eth0.2 nomaster
ip link set br-lan down
ip link delete br-lan type bridge

# まずは VLAN 1 を直接管理IFとして検証する。
ip addr replace <wired-management-ip>/<prefix> dev eth0.1
ip link set eth0.1 up
```

この時点で、Wi-Fi SSH と有線 SSH を別々に検証する。有線が通らない場合、
物理ケーブルがどちらの VLAN に出ているかを `ip -s link`、ARP neighbor、
switch FDB で確認する。Wi-Fi 経路を変更してはいけない。

### 永続設定と post-boot actor

`/etc/config/network` を変更するだけでは不十分なことがある。SBAT6 には
起動後段でネットワークを再構成する vendor script や recovery script が
存在し得るため、次を必ず検索する。

```sh
grep -R -n 'br-lan\|eth0\.1\|eth0\.2' \
  /etc/init.d /etc/rc.d /usr/sbin /sbin /lib 2>/dev/null
logread | grep -E 'br-lan|eth0\.1|netifd'
```

特に `S99*` のような後段 startup entry が、bridge を再作成し、IP を
`br-lan` に戻し、Dropbear を特定 interface に縛り直すことがある。見つけた
actor は bridge を作らず、以降の raw `eth0` 構成を適用するよう更新する。

`/etc/rc.d/S99name -> ../init.d/name` のような起動 entry は、必ず symlink
として archive に保存する。`cp` の宛先が既存 symlink の場合に追従すると、
古い内容の通常ファイルを archive に残してしまい、次回起動で最新の init
script を上書きする。archive 内で確認する。

```sh
tar -tvzf /data/<seed-archive>.tgz | grep S99name
# 先頭が l であり、../init.d/name を指すことを確認する。
```

## raw `eth0` への移行

### 物理 switch の確認

機種の switch utility では、次の read-only 情報が重要である。

```sh
switch vlan dump egtag
switch dump
ip -s link show dev eth0
```

実測例では、CPU port だけが tagged で、LAN member port は untagged だった。
CPU port を VLAN 1 の untagged member にし、CPU ingress の PVID を VLAN 1 に
設定すると、Linux の raw `eth0` が有線LANを受けられた。

**port番号・member map・PVID は機種固有である。** 他機の値をコピーしない。
現在の table を保存し、対象機上の `switch help` と `switch vlan dump egtag`
から正しい CPU port と port map を確定する。

概念的な変更は次の通りである。

```sh
# 例のみ。<...> は対象機の read-only 観測値から決める。
switch vlan set <fid> <lan-vid> <member-map> 0 0 <all-members-untagged-map>
switch vlan pvid <cpu-port> <lan-vid>

ip addr flush dev <old-vlan-if>
ip link set <old-vlan-if> down
ip addr replace <wired-management-ip>/<prefix> dev eth0
ip link set eth0 up
```

この実行時テストで、Wi-Fi SSH と raw `eth0` の有線SSHがともに成功した後だけ、
永続設定を更新する。

1. `network.lan.device` を `eth0` にする。
2. Linux VLAN interface を作る `switch` / `switch_vlan` UCI section を削除する。
3. late startup script で物理 switch の untagged table と `eth0` の管理IPを設定する。
4. Dropbear の `Interface` 固定値を削除し、firewall で管理元だけを許可する。
5. 更新した `network`、startup script、watchdog、Dropbear config、rc.d symlink を
   永続 seed archive に含める。

## SSH と firewall

Dropbear は `0.0.0.0:22` で待受し、SSHの到達元は firewall で最小化する。
allow rule を drop rule より前へ入れる。

```sh
iptables -C INPUT -i eth0 -s <management-host> -p tcp --dport 22 -j ACCEPT \
  2>/dev/null || iptables -I INPUT 1 -i eth0 -s <management-host> \
  -p tcp --dport 22 -j ACCEPT
iptables -C INPUT -i eth0 -p tcp --dport 22 -j DROP \
  2>/dev/null || iptables -I INPUT 2 -i eth0 -p tcp --dport 22 -j DROP
```

vendor firewall reload が遅れて起きる場合、同じ限定 rule を再確認する watchdog を
用意する。watchdog は `br-lan` や VLAN device を参照しないこと。

## 永続 seed archive

一部の SBAT6 構成では root overlay の変更だけでは不十分で、起動時に `/data`
の archive が展開・消費される。永続領域に seed archive を置き、必要なら
root cron などで次回展開用 archive を再配置する。

archive 更新後は、次を検証する。

```sh
tar -tzf /data/<seed-archive>.tgz
sha256sum /data/<seed-archive>.tgz /data/<next-boot-archive>.tgz
```

両 archive の hash が一致し、次の少なくとも一式が含まれることを確認する。

```text
etc/config/network
etc/config/dropbear
etc/rc.local
etc/init.d/<late-recovery-script>
etc/rc.d/S99<late-recovery-script>   # symlink
usr/sbin/<ssh-watchdog>
etc/init.d/<wifi-sta-service>
```

## 再起動テストと受入条件

各変更の後に一度だけでなく、vendor 後段が完了するまで待って確認する。
この系列では SSH 復帰まで数分を要することがある。

```sh
# Wi-Fi path
ssh -i <key> root@<sta-management-ip> 'id; ip -br addr; ip route'

# Wired raw-eth0 path
ssh -i <key> root@<wired-management-ip> 'id; ip -br addr; ip route'

# Target-side layout
ip link show br-lan && echo unexpected || true
ip link show eth0.1 && echo unexpected || true
ip -br addr show dev eth0
switch vlan dump egtag
```

受入条件は以下。

- Wi-Fi STA は association、DHCP、default route を保持する。
- Wi-Fi root SSH と有線 root SSH が別々に成功する。
- `br-lan` は不在。
- `eth0.1` / `eth0.2` は不在。
- `eth0` のみが有線管理IPを持つ。
- CPU側は untagged forwarding domain に所属する。
- seed archive の検証に成功する。
- 再起動後、後段 startup script 完了後にもすべて成立する。

## ロールバック

Wi-Fi SSHを維持したまま、保存済みの switch table、UCI network config、
startup script、seed archive を戻す。raw partition、boot metadata、factory/NV
領域を使う復旧は不要である。

概念例:

```sh
# 保存済みの元の switch member/tag map と PVID を復元する。
switch vlan set <original-arguments>
switch vlan pvid <cpu-port> <original-pvid>

# 保存済みの rootfs overlay 設定を戻し、必要なら next-boot archive を戻す。
cp /data/<backup>/network /etc/config/network
cp /data/<backup>/<late-recovery-script> /etc/init.d/<late-recovery-script>
cp /data/<backup>/<seed-archive> /data/<seed-archive>
sync
reboot
```

ロールバック値を推測で作らない。必ず変更前に保存した read-only 観測値と
設定バックアップから復元する。

## 帯域について

Linux bridge を削除しても、物理ポート間の帯域上限が必ず変わるとは限らない。
上限は switch fabric、SoC MAC、CPU port、driver offload、packet path によって
決まる。bridge/VLAN の有無と帯域を結論づけるには、同一ケーブル・同一peer・
同一MTUで、変更前後の双方向測定を分けて比較する。
