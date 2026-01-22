# CloudflareSpeedTest

-config 指定配置文件，第一次运行没有自动生成默认的config.json

思路：
顺序扫描给定的CIDR网段, 每次Num个, 能访问的放白名单，不能访问的放黑名单

下次再测试，拿一部分新ip，拿一部分白名单的ip，再配上上次的结果，三部分去测

想测下载速度必须手动指定下载的url，（你服务器的一个小文件，注意下载次数）

没做获取管理员权限，所以你不用管理员运行会写入hosts失败。

## 感谢项目

- https://github.com/XIU2/CloudflareSpeedTest
- https://github.com/Spedoske/CloudflareScanner


## License

The GPL-3.0 License.
