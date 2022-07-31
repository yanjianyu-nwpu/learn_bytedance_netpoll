# Socket in netpoll

## 0 introduction

   记录netpoll的是socket文件作用。

## 1 sys_keppalive_unix

```
// just support ipv4
func SetKeepAlive(fd, secs int) error {
    // open keep-alive
    if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
        return err
    }
    // tcp_keepalive_intvl
    if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, secs); err != nil {
        return err
    }
    // tcp_keepalive_probes
    // if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, 1); err != nil {
    //     return err
    // }
    // tcp_keepalive_time
    return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, secs)
}
```

    这里就是设置 SetKeepAlive tcp 。感觉设置定时keep-alive

## 2 sys_zerocopy_linux

    这里有两个函数，一个是设置零拷贝

```
func setZeroCopy(fd int) error {
    return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_ZEROCOPY, 1)
}

func setBlockZeroCopySend(fd int, sec, usec int64) error {
    return syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, SO_ZEROBLOCKTIMEO, &syscall.Timeval{
        Sec:  sec,
        Usec: usec,
    })
}
```

第一个函数感觉是设置 零拷贝 第二个设置非阻塞两零拷贝

## 3 sys_socketopt_linux

    这里感觉是设置ipv6的

```
func setDefaultSockopts(s, family, sotype int, ipv6only bool) error {
    if family == syscall.AF_INET6 && sotype != syscall.SOCK_RAW {
        // Allow both IP versions even if the OS default
        // is otherwise. Note that some operating systems
        // never admit this option.
        syscall.SetsockoptInt(s, syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, boolint(ipv6only))
    }

    // Allow broadcast.
    return os.NewSyscallError("setsockopt", syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1))
}
```

感觉做用不大

## 4 sys_sendmsg_linux.go

```
// sendmsg wraps the sendmsg system call.
// Must len(iovs) >= len
func sendmsg(fd int, bs [][]byte, ivs []syscall.Iovec, zerocopy bool) (n int, err error) {
    iovLen := iovecs(bs, ivs)
    if iovLen == 0 {
        return 0, nil
    }
    var msghdr = syscall.Msghdr{
        Iov:    &ivs[0],
        Iovlen: uint64(iovLen),
    }
    var flags uintptr
    if zerocopy {
        flags = MSG_ZEROCOPY
    }
    r, _, e := syscall.RawSyscall(syscall.SYS_SENDMSG, uintptr(fd), uintptr(unsafe.Pointer(&msghdr)), flags)
    resetIovecs(bs, ivs[:iovLen])
    if e != 0 {
        return int(r), syscall.Errno(e)
    }
    return int(r), nil
}
```

 这里就是包装了 sendmsg 系统调用

这里iovec 是linux为了提升磁盘数据到内存的效率，引入了io向量机制，io即struct iovec 在api接口在readv 和writev 使用。结构体比较简单

```
struct iovec{

      void *iov_base; /* pointer to the start of buffer */

      size_t iov_len; /* size of buffer in bytes */

}
```

在内核态中使用，并且有一个iovec向量依次读写，并且地址是连续的

这里读完感觉就是 把整个结果放到 缓冲区 然后发送出去

## 5 sys_exec.go

- socket rd
  
  ```
  func GetSysFdPairs() (r, w int) {
      fds, _ := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
      return fds[0], fds[1]
  }
  
  // setTCPNoDelay set the TCP_NODELAY flag on socket
  func setTCPNoDelay(fd int, b bool) (err error) {
      return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, boolint(b))
  }
  ```
  
  设置socket pair

    设置no delay socket

- sysSocket 可以更加简单的设置socket 有些协议栈，so type 还有 协议proto
  
  ```
  
  ```

    就是简单的

- wirtev 和 readv 这里就是感觉把复制过去

```

```

把结果拷过去

- 然后使用iovecs

```
// TODO: read from sysconf(_SC_IOV_MAX)? The Linux default is
//  1024 and this seems conservative enough for now. Darwin's
//  UIO_MAXIOV also seems to be 1024.
func iovecs(bs [][]byte, ivs []syscall.Iovec) (iovLen int) {
    for i := 0; i < len(bs); i++ {
        chunk := bs[i]
        if len(chunk) == 0 {
            continue
        }
        ivs[iovLen].Base = &chunk[0]
        ivs[iovLen].SetLen(len(chunk))
        iovLen++
    }
    return iovLen
}

func resetIovecs(bs [][]byte, ivs []syscall.Iovec) {
    for i := 0; i < len(bs); i++ {
        bs[i] = nil
    }
    for i := 0; i < len(ivs); i++ {
        ivs[i].Base = nil
    }
}
```

就是拷贝和复原缓冲区

## 6 sys_Epoll_linux

- 首先是 epollevent 这里 应该是记录 events fd 然后一个size ptr

```
type epollevent struct {
    events uint32
    data   [8]byte // unaligned uintptr
}
```

- 然后注册文件描述符

```
// EpollCtl implements epoll_ctl.
func EpollCtl(epfd int, op int, fd int, event *epollevent) (err error) {
    _, _, err = syscall.RawSyscall6(syscall.SYS_EPOLL_CTL, uintptr(epfd), uintptr(op), uintptr(fd), uintptr(unsafe.Pointer(event)), 0, 0)
    if err == syscall.Errno(0) {
        err = nil
    }
    return err
}
```

- 然后就是 epollWait

```
// EpollWait implements epoll_wait.
func EpollWait(epfd int, events []epollevent, msec int) (n int, err error) {
    var r0 uintptr
    var _p0 = unsafe.Pointer(&events[0])
    if msec == 0 {
        r0, _, err = syscall.RawSyscall6(syscall.SYS_EPOLL_WAIT, uintptr(epfd), uintptr(_p0), uintptr(len(events)), 0, 0, 0)
    } else {
        r0, _, err = syscall.Syscall6(syscall.SYS_EPOLL_WAIT, uintptr(epfd), uintptr(_p0), uintptr(len(events)), uintptr(msec), 0, 0)
    }
    if err == syscall.Errno(0) {
        err = nil
    }
    return int(r0), err
}
```

- 大概如上

## 7 poll_race_linux.go

```
func openDefaultPoll() *defaultPoll {
	var poll = defaultPoll{}
	poll.buf = make([]byte, 8)
	var p, err = syscall.EpollCreate1(0)
	if err != nil {
		panic(err)
	}
	poll.fd = p
	var r0, _, e0 = syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(p)
		panic(err)
	}
	poll.wfd = int(r0)
	poll.Control(&FDOperator{FD: poll.wfd}, PollReadable)
	return &poll
}
```

这里召唤defaultPoll 结构体，



```
// Wait implements Poll.
func (p *defaultPoll) Wait() (err error) {
	// init
	var caps, msec, n = barriercap, -1, 0
	p.reset(128, caps)
	// wait
	for {
		if n == p.size && p.size < 128*1024 {
			p.reset(p.size<<1, caps)
		}
		n, err = syscall.EpollWait(p.fd, p.events, msec)
		if err != nil && err != syscall.EINTR {
			return err
		}
		if n <= 0 {
			msec = -1
			runtime.Gosched()
			continue
		}
		msec = 0
		if p.handler(p.events[:n]) {
			return nil
		}
	}
}
```

这里epoll wait 这里也非常简单

```
// Wait implements Poll.
func (p *defaultPoll) Wait() (err error) {
	// init
	var caps, msec, n = barriercap, -1, 0
	p.reset(128, caps)
	// wait
	for {
		if n == p.size && p.size < 128*1024 {
			p.reset(p.size<<1, caps)
		}
		n, err = syscall.EpollWait(p.fd, p.events, msec)
		if err != nil && err != syscall.EINTR {
			return err
		}
		if n <= 0 {
			msec = -1
			runtime.Gosched()
			continue
		}
		msec = 0
		if p.handler(p.events[:n]) {
			return nil
		}
	}
}
```



这里非常重要的，这里如果是 超时唤醒的 runtime.Gosched

- handler 函数
  
  ```
  func (p *defaultPoll) handler(events []syscall.EpollEvent) (closed bool) {
  	for i := range events {
  		var fd = int(events[i].Fd)
  		// trigger or exit gracefully
  		if fd == p.wfd {
  			// must clean trigger first
  			syscall.Read(p.wfd, p.buf)
  			atomic.StoreUint32(&p.trigger, 0)
  			// if closed & exit
  			if p.buf[0] > 0 {
  				syscall.Close(p.wfd)
  				syscall.Close(p.fd)
  				return true
  			}
  			continue
  		}
  		tmp, ok := p.m.Load(fd)
  		if !ok {
  			continue
  		}
  		operator := tmp.(*FDOperator)
  		if !operator.do() {
  			continue
  		}
  
  		evt := events[i].Events
  		// check poll in
  		if evt&syscall.EPOLLIN != 0 {
  			if operator.OnRead != nil {
  				// for non-connection
  				operator.OnRead(p)
  			} else {
  				// for connection
  				var bs = operator.Inputs(p.barriers[i].bs)
  				if len(bs) > 0 {
  					var n, err = readv(operator.FD, bs, p.barriers[i].ivs)
  					operator.InputAck(n)
  					if err != nil && err != syscall.EAGAIN && err != syscall.EINTR {
  						log.Printf("readv(fd=%d) failed: %s", operator.FD, err.Error())
  						p.appendHup(operator)
  						continue
  					}
  				}
  			}
  		}
  
  		// check hup
  		if evt&(syscall.EPOLLHUP|syscall.EPOLLRDHUP) != 0 {
  			p.appendHup(operator)
  			continue
  		}
  		if evt&syscall.EPOLLERR != 0 {
  			// Under block-zerocopy, the kernel may give an error callback, which is not a real error, just an EAGAIN.
  			// So here we need to check this error, if it is EAGAIN then do nothing, otherwise still mark as hup.
  			if _, _, _, _, err := syscall.Recvmsg(operator.FD, nil, nil, syscall.MSG_ERRQUEUE); err != syscall.EAGAIN {
  				p.appendHup(operator)
  				continue
  			}
  		}
  
  		// check poll out
  		if evt&syscall.EPOLLOUT != 0 {
  			if operator.OnWrite != nil {
  				// for non-connection
  				operator.OnWrite(p)
  			} else {
  				// for connection
  				var bs, supportZeroCopy = operator.Outputs(p.barriers[i].bs)
  				if len(bs) > 0 {
  					// TODO: Let the upper layer pass in whether to use ZeroCopy.
  					var n, err = sendmsg(operator.FD, bs, p.barriers[i].ivs, false && supportZeroCopy)
  					operator.OutputAck(n)
  					if err != nil && err != syscall.EAGAIN {
  						log.Printf("sendmsg(fd=%d) failed: %s", operator.FD, err.Error())
  						p.appendHup(operator)
  						continue
  					}
  				}
  			}
  		}
  		operator.done()
  	}
  	// hup conns together to avoid blocking the poll.
  	p.detaches()
  	return false
  }
  ```
  
  
  
  这里就是处理读写事件 最后连接实践 分离处理，不要阻塞epoll

## 8 poll_manager.go

这里是结构体

```
type manager struct {
	NumLoops int
	balance  loadbalance // load balancing method
	polls    []Poll      // all the polls
}
```

然后init 

```
func init() {
	var loops = runtime.GOMAXPROCS(0)/20 + 1
	pollmanager = &manager{}
	pollmanager.SetLoadBalance(RoundRobin)
	pollmanager.SetNumLoops(loops)
}
```

设置

```
// Run all pollers.
func (m *manager) Run() error {
	// new poll to fill delta.
	for idx := len(m.polls); idx < m.NumLoops; idx++ {
		var poll = openPoll()
		m.polls = append(m.polls, poll)
		go poll.Wait()
	}
	// LoadBalance must be set before calling Run, otherwise it will panic.
	m.balance.Rebalance(m.polls)
	return nil
}
```

非常简单
