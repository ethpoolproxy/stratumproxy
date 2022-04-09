#!/bin/bash
[[ $(id -u) != 0 ]] && echo -e "请使用root权限运行安装脚本" && exit 1

cmd="apt-get"
if [[ $(command -v apt-get) || $(command -v yum) ]] && [[ $(command -v systemctl) ]]; then
    if [[ $(command -v yum) ]]; then
        cmd="yum"
    fi
else
    echo "此脚本不支持该系统" && exit 1
fi

install() {
    if [ -f "/usr/bin/stratumproxy" ]; then
        echo -e "您已安装了该软件，如果确定没有安装，请使用此脚本的卸载功能后重新安装" && exit 1
    fi
    if pgrep stratumproxy; then
        echo -e "检测到您已启动了 /usr/bin/stratumproxy，请关闭后再安装！" && exit 1
    fi

    $cmd update -y
    $cmd install curl wget -y
    mkdir /etc/stratumproxy

    echo "请选择版本"
    echo "  1、v1.3.0 | 代号 [Rinako]"
    read -p "$(echo -e "请输入[1]：")" choose
    case $choose in
    1)
        wget https://github.com/ethpoolproxy/stratumproxy/releases/download/v1.3.0/stratumproxy_v1.3.0 -O /usr/bin/stratumproxy
        wget https://raw.githubusercontent.com/ethpoolproxy/stratumproxy/stratumproxy.service -O /etc/systemd/system/stratumproxy.service
        ;;
    *)
        echo "请输入正确的数字"
        ;;
    esac

    chmod +x /usr/bin/stratumproxy

    echo "正在启动..."
    systemctl daemon-reload
    systemctl enable --now stratumproxy
    sleep 2s
    journalctl --unit=stratumproxy --no-tail --lines=10
    echo "安装结束!"
}

uninstall() {
    read -p "是否确认删除 StratumProxy [yes/no]：" flag
    if [ -z $flag ]; then
        echo "输入错误" && exit 1
    else
        if [ "$flag" = "yes" -o "$flag" = "ye" -o "$flag" = "y" ]; then
            systemctl disable --now stratumproxy
            rm -rf /etc/systemd/system/stratumproxy.service
            rm -rf /usr/bin/stratumproxy
            rm -rf /etc/stratumproxy
            systemctl daemon-reload
            echo "卸载 StratumProxy 成功"
        fi
    fi
}

start() {
    systemctl enable --now stratumproxy
    sleep 2s
    journalctl --unit=stratumproxy --no-tail --lines=10

    echo "StratumProxy 已启动"
}

restart() {
    systemctl restart stratumproxy
    sleep 2s
    journalctl --unit=stratumproxy --no-tail --lines=10

    echo "StratumProxy 重新启动成功"
}

stop() {
    systemctl stop stratumproxy
    echo "StratumProxy 已停止"
}

show_log(){
    echo -n "最近的 100 行日志: "
    journalctl --unit=stratumproxy --no-tail --lines=100
}

check_limit(){
    echo -n "当前连接数限制：102400"
}

echo "============================ StratumProxy ============================"
echo "  1、安装(安装到 程序:/usr/bin/stratumproxy 配置文件:/etc/stratumproxy)"
echo "  2、卸载(更新请先卸载，请注意: 配置文件不兼容 需要重新配置)"
echo "  3、启动"
echo "  4、重启"
echo "  5、停止"
echo "  6、查看最近的 100 行日志"
echo "  7、查看软件连接数限制"
echo "======================================================================"
read -p "$(echo -e "请选择[1-6]：")" choose
case $choose in
1)
    install
    ;;
2)
    uninstall
    ;;
3)
    start
    ;;
4)
    restart
    ;;
5)
    stop
    ;;
6)
    show_log
    ;;
7)
    check_limit
    ;;
*)
    echo "输入错误，请重新输入！"
    ;;
esac
