#### 配置导向型低交互蜜罐服务器

目前支持将`SSH`部署在指定非特权端口上作为蜜罐，记录攻击者IP、证书公钥、攻击时间、使用的用户名与弱口令等信息、支持热重载除数据库以外的配置信息。
后续将在使用容器进一步隔离服务程序，并加入特权端口、systemctl的支持，此外打算引入文本模型来增强交互能力。

#### 参考在线文档

- [golang.halfiisland.com/community/pkgs/orm/gorm.html#外键](https://golang.halfiisland.com/community/pkgs/orm/gorm.html#%E9%92%A9%E5%AD%90)
- [?](https://bg6cq.github.io/ITTS/)


#### TODOs

- 如果可能，配置[oss-fuzz](https://google.github.io/oss-fuzz/getting-started/new-project-guide/)