# introduction

## 0 background

    Go 原生自带的 net 库是阻塞的I/O api, 所以rpc框架只能 per connect per goroutine这样的设计。此外net.Conn没有提供连接活性的API，所以不好设计连接池，池中的失效连接

    netpoll 由字节跳动开发的高性能 NIO 网络库，专注RPC场景。



## 1 feature

- [LinkBuffer](https://github.com/cloudwego/netpoll/blob/develop/nocopy_linkbuffer.go) 提供可以流式读写的 nocopy API
- [gopool](https://github.com/bytedance/gopkg/tree/develop/util/gopool) 提供高性能的 goroutine 池
- [mcache](https://github.com/bytedance/gopkg/tree/develop/lang/mcache) 提供高效的内存复用
- `IsActive` 支持检查连接是否存活
- `Dialer` 支持构建 client
- `EventLoop` 支持构建 server
- 支持 TCP，Unix Domain Socket
- 支持 Linux，macOS（操作系统）
