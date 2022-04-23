#!/bin/python

import socket
import sys
import ssl
import time
import json
import random
import threading

if len(sys.argv) - 1 != 8:
    print("./fake-miner-eth.py [SSL|TCP] [IP] [Port] [ShareDelay] [HashratePerGPU(MHs)] [GPU] [Wallet] [Worker]")
    exit(-1)
Ssl = sys.argv[1] == "ssl"
Ip = sys.argv[2]
Port = sys.argv[3]
ShareDelay = sys.argv[4]
Hashrate = sys.argv[5]
Gpu = sys.argv[6]
Wallet = sys.argv[7]
Worker = sys.argv[8]

Jobs = []
shareSent = 0
conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

SocketLock = threading.Lock()
def WriteStr(msg):
    global conn
    SocketLock.acquire()
    conn.sendall(str.encode(msg + "\n"))
    SocketLock.release()

def readLine():
    return conn.recv(1024).decode()

def Mining():
    global Jobs
    while True:
        time.sleep(float(ShareDelay) + random.randint(-int(ShareDelay)/2, int(ShareDelay)/2))
        if len(Jobs) == 0:
            continue
        jobIndex = random.randint(0, len(Jobs) - 1)
        WriteStr("{\"id\": 114514,\"method\": \"eth_submitWork\",\"worker\":\"" + Worker + "\",\"params\": [\"0x114514\",\"" + Jobs[jobIndex] + "\",\"0x114514\"]}")
        del Jobs[jobIndex]

def ProcessPacket(packet):
    global conn
    global shareSent
    global Jobs

    # eth_submitWork result ID: 114514
    # result for job
    if packet["id"] == 114514:
        print("ethash - Share Submit #" + str(shareSent))

    # Send Hashrate
    if shareSent % 20 == 0:
        WriteStr("{\"id\":9,\"jsonrpc\":\"2.0\",\"method\":\"eth_submitHashrate\",\"params\":[\"" + hex(int(Hashrate) * int(Gpu) * 1000000) + "\",\"114514\"],\"worker\":\"" + Worker + "\"}")

    # job packet
    if type(packet["result"]) is list:
        print("ethash - New job: " + packet["result"][0][:8])
        if len(Jobs) >= 20:
            del Jobs[0]
        Jobs.append(packet["result"][0])
        shareSent = shareSent + 1

    # eth_submitHashrate ID: 1919810
    #if packet["id"] == 1919810

if __name__ == '__main__':
    conn.setblocking(True)
    if Ssl:
        sslCtx = ssl.create_default_context()
        sslCtx.check_hostname = False
        sslCtx.verify_mode = ssl.CERT_NONE
        conn = sslCtx.wrap_socket(conn)

    print("Connect to [" + Ip + ":" + Port + "]")
    conn.connect((Ip, int(Port)))
    time.sleep(1)

    # login
    WriteStr("{\"compact\":true,\"id\":1,\"method\":\"eth_submitLogin\",\"params\":[\"" + Wallet + "\",\"\"],\"worker\":\"" + Worker + "\"}")
    loginResponse = readLine()
    if "false" in loginResponse:
        print("Login failed: " + loginResponse)
        exit(-1)

    # Get work
    WriteStr("{\"method\":\"eth_getWork\",\"id\":5,\"params\":[]}")

    for i in range(0, int(Gpu)):
        t = threading.Thread(target=Mining, args=())
        t.start()

    # server send
    while True:
        ProcessPacket(json.loads(readLine()))
