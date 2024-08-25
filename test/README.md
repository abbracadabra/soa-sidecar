1、运行sidecar程序  

2、运行nacos，用作注册中心

3、启动业务服务，这里我们开启一个http echo服务和一个grpc echo服务，每个服务启动完后调下面api：    
- 启动代理(optional)：如果想让sidecar代理我的服务，就调sidecar的rest api`/startInboundProxy`，一般由业务服务自己调    
- 注册服务：服务想被代理就注册代理地址，反之注册服务自己地址，这里有三种注册方式都行： 1:服务直连注册中心 2：注册请求经过sidecar转发到注册中心 3:sidecar开了rest api来转发注册请求，这样简化了业务侧的注册参数，底层换注册中心业务也不用改

4、让服务A、B互调，看是否走通
- 请求域名的格式为：`{service}.soa.proxy.net`，其中`{service}`是目标服务名，sidecar根据`{service}`路由  
- 出口流量要打到sidecar，因此需要把域名解析到sidecar，因此需要修改本地host文件，或者修改公司域名服务器  