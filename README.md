# mirror-proxy

使用代理下载文件

## 使用

启动服务

```bash
./mirror-proxy -add=192.168.2.111:8888
```

下载文件

```bash
wget --content-disposition http://192.168.2.111:8888/url='https://github.com/jimyag/parquet-tools/releases/download/v1.1.1/parquet-tools-v1.1.1-linux-amd64.tar.gz'
```
