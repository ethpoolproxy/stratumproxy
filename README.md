# StratumProxy
<a href="https://t.me/StratumProxy">StratumProxy TG 交流群</a>
![webui.jpg](webui.jpg)  

## 更新日志

## Windows 直接下载运行 
https://github.com/ethpoolproxy/stratumproxy/releases

## Linux一键安装

```bash
bash <(curl -s -L https://raw.githubusercontent.com/ethpoolproxy/stratumproxy/master/install.sh)
```

---

### 查看运行情况
```bash
systemctl status stratumproxy
```
或者使用脚本查看日志

---
## Linux手动安装
```bash
wget https://github.com/ethpoolproxy/stratumproxy/releases/download/v1.3.1/stratumproxy_v1.3.1 -O /usr/bin/stratumproxy
wget https://raw.githubusercontent.com/ethpoolproxy/stratumproxy/stratumproxy.service -O /etc/systemd/system/stratumproxy.service
systemctl daemon-reload
systemctl enable --now stratumproxy
```

## 重要说明

```bigquery
开发者费用
本软件如果您开启了抽水则为0.3%的开发者费用,如果您不开启抽水,则没有开发者费用,可以自行抓包查看
推荐使用腾讯云香港节点,flexpool和ethermine都可以到50ms左右,延迟率在0.5%-0.9%之间
该软件系统占用极小,开最便宜的云服务器即可（不要使用轻量服务器,轻量网络极差）
```
