### soa-sidecar, 可能soa-proxy更贴切, soa-mesh听起来更酷？？
soa基础设施（服务路由、负载均衡、连接池、限流、熔断、调用鉴权、泳道、打点、注册、...）一般以sdk形式集成在业务应用，soa-sidecar把soa设施放在服务边车，好处就是不用每个语言写一套sdk，soa升级不需要业务服务升级，等等...。业务侧不需要sdk或只需要thin sdk就能接入  
  
- 支持协议：grpc,http1/2,tls  
- 支持出流量、入流量代理  
- 支持透明代理  
- 不仅仅是sidecar，你可以当成普通服务部署（sidecarless？）  

### todo:  
- thin sdk
- soa-sidecar 灰度升级（thin sdk 或者 socket fd迁移）  
- 服务端自适应限流摘流
- 客户端熔断
- 流量拓扑
- uds
- di:uber-go/dig

### 使用示例
1、运行sidecar程序`go run ./cmd -c ./test/testConfig.yaml`  

2、运行nacos，用作注册中心  

3、启动业务服务，这里我们开启一个http echo服务和一个grpc echo服务，每个服务启动完后调下面api：
- 启动代理(optional)：如果想让sidecar代理入口流量，就调sidecar的rest api`/startInboundProxy`，一般由业务服务自己调
- 注册服务：服务想被代理就注册代理地址，反之注册服务自己地址，这里有三种注册方式都行：  
1：服务直连注册中心  
2：把注册中心当成一个upstream cluster，让sidecar把注册请求转发到注册中心   
3：调sidecar的rest api`/exportService`来转发注册请求，这样简化了业务侧的注册参数，底层换注册中心业务也不用改

4、让服务A、B互调
- 请求域名的格式为：`{service}.soa.proxy.net`，其中`{service}`是目标服务名，sidecar根据`{service}`路由
- 出口流量要打到sidecar，因此需要把域名解析到sidecar，因此需要修改本地host文件，或者修改公司域名服务器  