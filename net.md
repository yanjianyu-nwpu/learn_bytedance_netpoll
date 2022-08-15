## Golang net

## 0 Introduction

网络io 还有 icp ip udp    domain name(域名)



感觉连接可以同通过dial 来搞

客户端代码

```
conn, err := net.Dial("tcp", "google.com:80")
if err != nil {
	// handle error
}
fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
status, err := bufio.NewReader(conn).ReadString('\n')
// ...
```

可以拿到 连接conn

 服务端代码

```
ln, err := net.Listen("tcp", ":8080")
if err != nil {
	// handle error
}
for {
	conn, err := ln.Accept()
	if err != nil {
		// handle error
		continue
	}
	go handleConnection(conn)
}
```

直接读取连接的函数
