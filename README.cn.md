# hls-get
   命令行下面的HLS（M3U8）下载工具
   
## 使用场景：

   * 简单模式: 以命令行参数的方式下载一个或多个的URL；
   * Redis列表: 通过Redis获取下载URL列表；
   * MySQL列表: 通过MySQL获取下载列表.

## 使用方法：

    hls-get [OPTIONS,...] [URL1,URL2,...]

### 参数
- `c` 字符串
      指定使用一个配置文件配置下面的各个参数。默认为空（即不使用配置文件）。
- `O` 字符串
      下载文件保存路径（默认当前路径"."）
- `CO` 数字
      并发任务数量（默认为5）
- `L` 字符串
      日志输出文件名。默认为空则输出到标准输出。
- `V` 字符串
      日志输出级别。默认“INFO”。
- `R` 数字
      错误重试次数。默认1次。
- `RR` 字符串
      重定向服务器URL。默认空。
- `S`  布尔值
      是否跳过已存在的文件。默认false。
- `SZ` 布尔值
      校验已存在的文件大小是否与源文件一样才跳过。默认false。
- `TO` 数字
      单个请求超时时间（单位：秒，默认20秒）
- `TT` 数字
      一共下载多少个URL。默认0即为无限制。
- `UA` 字符串
      UserAgent. (default "hls-get v0.9.10")
- `SR` 字符串
      切片文件名重写规则，默认为空，即原样复制。*暂未实现*
- `PR` 字符串
      下载保存路径重写规则。默认为空即原样复制。
      重写规则由一个类似`sed`的软件包实现，具体可以参阅：https://github.com/rwtodd/sed-go
      大部分情况下类似`sed`用法，但有一点小差别：
          
| Go-sed          |  Traditional RE   | Notes                             |
| --------------- | ----------------- | --------------------------------- |
|  s/a(bc*)d/$1/g |  s/a\\(bc*\\)d/\1/g | Don't escape (); Use $1, $2, etc. |
|  s/(?s).//      |  s/.//            | If you want dot to match \n, use (?s) flag.  |
      
- `M` 字符串
      指定下载列表的获取方式：mysql或redis；默认为空即从命令行参数传入下载地址；

- `MD` 字符串
      MySQL数据库名称，默认为："hlsgetdb"；
- `MH` 字符串
      MySQL数据库服务器，默认为："localhost"；
- `MN` 字符串
      MySQL用户名，默认为："root"；
- `MP` 数字
      MySQL数据库端口，默认为：3306；
- `MT` 字符串
      MySQL数据表，默认为："hlsget_downloads"；
      数据表的格式参照后面描述，也可以使用“-MS” 命令输出；
- `MW` 字符串
      MySQL数据库密码；


- `RH` 字符串
      Redis服务器地址，默认："localhost"；
- `RP` 数字
      Redis服务器端口，默认：6379
- `RW` 字符串
      Redis密码，默认为空即不需要密码；
- `RD` 数字
      Redis数据库，默认为0；
- `RK` 字符串
      Redis里列表的KEY名称，默认为："HLSGET_DOWNLOADS"；

### MySQL数据表结构:

  下面是hsl-get使用MySQL作为下载列表是数据表的结构介绍。
  
  - `url` 字段是需要下载的URL地址；
  
  - `dest` 是下载后存放的路径，该字段由hls-get填写；
  
  - `status` 下载的状态，0 表示等待下载, 1 表示正在下载, 2 为下载成功, <0 表示下载失败
  
  - `ret_code` 和 `ret_msg` 两个字段记录下载的结果。
 
  - hlsget_downloads数据表的结构SQL：
  
          DROP TABLE IF EXISTS `hlsget_downloads`;
          CREATE TABLE `hlsget_downloads` (
            `id` int(11) NOT NULL AUTO_INCREMENT,
            `url` varchar(256) NOT NULL,
            `status` int(11) NOT NULL DEFAULT '0',
            `dest` varchar(256) DEFAULT NULL,
            `ret_code` int(11) DEFAULT '0',
            `ret_msg` varchar(128) DEFAULT NULL,
            PRIMARY KEY (`id`),
            UNIQUE KEY `url` (`url`)
          ) ENGINE=InnoDB AUTO_INCREMENT=393211 DEFAULT CHARSET=latin1;
 
  - 下面是一个简单的从epgdb_vod.pulish_movie灌入下载地址的脚本：
  
        INSERT INTO hlsgetdb.hlsget_downloads (url, ret_code) SELECT `guid`, 0 FROM epgdb_vod.publish_movie WHERE `guid` <> "";

### Redis的数据结构：

Redis中下载地址列表需要是一个LIST，假设下载列表的KEY为`DOWNLOADS`，那么`hls-get`会将正在下载的URL放在`DOWNLOADS_RUNNING`里面，下载成功的放在`DOWNLOADS_SUCCESS`里面，而失败则会放在`DOWNLOADS_FAILED`里面。
注意： DOWNLOADS 是一个LIST， 而 `DOWNLOADS_RUNNING`、`DOWNLOADS_SUCCESS`和`DOWNLOADS_FAILED`则是三个HASHMAP，结构类似MySQL的数据表定义。
  
## 使用MySQL的样例：

        ./hls-get -O=/data/videos -C 10 -M=mysql -MW=root -TO=10 -TT=100 -S -R=3 -PR='s/\/vds[0-9]+\/data[0-9]+\/(.*)/$1/g' -RR='http://videoha.example.org/redirect?url=%s'
    
- `-C 10` 10个并发
- `-M=mysql` 使用MySQL
- `-MW=root` 密码为“root”
- `-TO=10` 超时10秒
- `-TT=100` 一共下载100个链接
- `-S` 跳过存在的文件
- `-R=3` 失败重试3次
- `-PR='s/\/vds[0-9]+\/data[0-9]+\/(.*)/$1/g'` 输出路径重写
- `-RR='http://videoha.example.org/redirect?url=%s'` 重定向URL， 从数据库取出的每个URL会填充到`%s`然后再发起下载请求

## 配置文件样例：

    ## This is an example of config file for hls-get
    ## The format of config is TOML (like ini file)
    # Output string
    output="./"
    # PathRewrite string
    path_rewrite="s/\\/vds[0-9]+\\/data[0-9]+\\/(.*)/$1/g"
    # SegmentRewrite string
    segment_rewrite=""
    # UserAgent string
    user_agent="HLS-GET"
    # LogFile string
    log_file=""
    # LogLevel string
    log_level="DEBUG"
    # Retries int
    retries=3
    # Skip bool
    skip=true
    # Redirect string
    redirect="http://videoha.example.org/redirect?url=%s"
    # Concurrent int
    concurrent=5
    # Timeout int
    timeout=20
    # Total int64
    total=0
    # Mode string
    mode="mysql"
    
    # Redis RedisConfig
    [redis]
    host="127.0.0.1"
    port=6379
    password=""
    db=1
    key="DOWNLOAD_MOVIES"
    
    # MySQL MySQLConfig
    [mysql]
    host="127.0.0.1"
    port=3306
    db="hlsget_db"
    table="download_movies"