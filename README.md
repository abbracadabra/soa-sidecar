### soa-sidecar  
soa基础设施（服务路由、负载均衡、连接池、限流、熔断、调用鉴权、泳道、打点、注册、...）一般以sdk形式集成在业务应用，soa-sidecar把soa设施放在服务边车，好处就是不用每个语言写一套sdk，soa升级不需要业务服务升级，等等...。业务侧不需要sdk或只需要thin sdk就能接入  
  
- 支持协议：grpc,http1/2,tls  
- 支持出流量、入流量代理  
- 支持透明代理  
- 支持sidecarless 模式  
  
  
  
todo:  
- thin sdk
- soa-sidecar 灰度升级（thin sdk 或者 socket fd迁移）  


