# wallet

> 虚拟币钱包服务，转账/提现/充值/归集
> 
> 计划支持：比特币、以太坊（ERC20）、波场（TRC20），币安（BEP20）
> 
> 完全实现与业务服务隔离，使用http服务相互调用

# 接口

`script/api.md`

# 下载-打包

```shell
# 拉取代码
$ git clone https://github.com/lmxdawn/wallet.git
$ cd wallet

# 打包 (-tags "doc") 可选，加上可以运行swagger
$ go build [-tags "doc"]

# 运行
$ wallet -c config/config-example.yml

```
> 启动后访问： `http://localhost:10009/swagger/index.html`

# Swagger

> 把 swag cmd 包下载 `go get -u github.com/swaggo/swag/cmd/swag`

> 这时会在 bin 目录下生成一个 `swag.exe` ，把这个执行文件放到 `$GOPATH/bin` 下面

> 执行 `swag init` 注意，一定要和main.go处于同一级目录

> 启动时加上 `-tags "doc"` 才会启动swagger。 这里主要为了正式环境去掉 swagger，这样整体编译的包小一些

> 启动后访问： `http://ip:prot/swagger/index.html`

# 第三方库依赖

> log 日志 `github.com/rs/zerolog`

> 命令行工具 `github.com/urfave/cli`

> 配置文件 `github.com/jinzhu/configor`

# 环境依赖

> go 1.16+

> Redis 3

> MySQL 5.7

# 其它

> `script/Generate MyPOJOs.groovy` 生成数据库Model

# 合约相关
> `solcjs.cmd --version` 查看版本
> 
> `solcjs.cmd --abi erc20.sol`
> 
> `abigen --abi=erc20_sol_IERC20.abi --pkg=eth --out=erc20.go`

# 吃鸡地址
> 0xDfdf53447cA55820Ec2B3dE9EA707A31579F5c0F
> 
> 定制开发请联系：https://t.me/aa333555

# 准备
要实现这些功能首先得摸清楚我们需要完成些什么东西

1. 获取最新区块
2. 获取区块内部的交易记录
3. 通过交易哈希获取交易的完成状态
4. 获取某个地址的余额
5. 创建一个地址
6. 签名并发送luo交易
7. 定义接口如下
```go
type Worker interface {
    getNowBlockNum() (uint64, error)
    getTransaction(uint64) ([]types.Transaction, uint64, error)
    getTransactionReceipt(*types.Transaction) error
    getBalance(address string) (*big.Int, error)
    createWallet() (*types.Wallet, error)
    sendTransaction(string, string, *big.Int) (string, error)
}
```
# 实现
> 创建一个地址后把地址和私钥保存下来
## 进
通过一个无限循环的服务不停的去获取最新块的交易数据，并且把交易数据都一一验证是否完成
，这里判断数据的接收地址（to）是否属于本服务创建的钱包地址，如果是本服务的创建过的地址则判断为充值成功，**（这时逻辑服务里面需要做交易哈希做幂等）**
## 出
用户发起一笔提出操作，用户发起提出时通过服务配置的私钥来打包并签名luo交易。（私钥转到用户输入的提出地址），这里把提交的luo交易的哈希记录到服务
通过一个无限循环的服务不停的去获取最新块的交易数据，并且把交易数据都一一验证是否完成
，这里判断交易数据的哈希是否存在于服务，如果存在则处理**（这时逻辑服务里面需要做交易哈希做幂等）**
## 归集
通过定期循环服务创建的地址去转账到服务配置的归集地址里面，这里需要注意归集数量的限制，当满足固定的数量时才去归集（减少gas费）

# 一个简单的示例

github地址： [golang 实现加密货币的充值/提现/归集服务](https://github.com/lmxdawn/wallet)
