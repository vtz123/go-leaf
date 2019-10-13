
**1. 两种模式**  
 
- snowflake模式
 图1：![leaf1.JPG](https://i.loli.net/2019/10/13/cr98SJtgfYsUjGy.jpg)
            
        41位作为毫秒数，可以表示69年的时间( 1L<<41 /(365 *24 *3600 *1000 )
        10位表示机器id，也可以采用 数据id + 机器id
        12位自增序列号，需上锁

- segment模式
        
		使用代理server批量(step决定数量大小)获取，减轻数据库读写压力
        各业务发号需求用tag字段区分，各类业务获取相互隔离
        
	
   
    leaf动态调整step
		
	    根据单位时间内并发量大小动态变化，服务QPS * 号段更新周期T = 号段长度L
	    如果更新周期为10min,那么即使DB跌宕，也能持续发号10-20min。

	下一次号段nextstep长度调整策略:
        
 	- T < 15min, nextstep = step * 2
    - 15min < T < 30min, nextstep = step
    - T > 30min, nextstep = step / 2
    

	双buffer优化：
    	
	  	当号段消耗完时，这期间从DB取回号段，若并发量过大或者DB网络、性能不稳定，会造成发号阻塞。
		为使得发号过程无阻塞，异步提前将下一个号段加载到内存中，而不必等到号码用尽再从DB中取。
图2

 **2. 请求方式**

 -  http 调用
 	
		cd go-leaf
		go run main.go
		snowflake模式：//http://localhost:8000/alloc-snowflake?machineid=11
		segment模式  ：//http://localhost:8000/alloc-segment?biz_tag=test

 -  grpc 调用
 
 		cd  go-leaf/grpc
		go run server.go
		cd go-leaf-example
		go run client.go

 	需根据.env文件配置好mysql数据库连接
 
 **3. segment模式监控**

 图3：
    


   提供Web层的内存数据映射界面，可以实时看到所有号段下发状态。比如每个号段双buffer的使用情况，id下发到那个位置等。
            
        
