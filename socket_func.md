# Socket in netpoll

## 0 introduction

   记录netpoll的是socket文件作用。

## 1 sys_keppalive_unix

    这里就是设置 SetKeepAlive tcp 。感觉就是

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
