## 更新日志

主要更新:
- 添加以下oid<br>
    SM3WithSM2 1.2.156.10197.1.501<br>
    SHA1WithSM2 1.2.156.10197.1.502<br>
    SHA256WithSM2 1.2.156.10197.1.503<br>

- x509生成的证书如今可以使用SM3作为hash算法

- 引入了以下hash算法
    RIPEMD160<br>
    SHA3_256<br>
    SHA3_384<br>
    SHA3_512<br>
    SHA3_SM3<br>
  用户需要自己安装golang.org/x/crypto
