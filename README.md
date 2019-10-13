思路参考 Leaf：美团分布式ID生成服务开源https://tech.meituan.com/2019/03/07/open-source-project-leaf.html
项目参考 美团开源Java版leaf  https://github.com/Meituan-Dianping/Leaf
测试：
  - http基准测试， 测试工具：apache bench
    命令1: 
    go-leaf: ab -n 30000 -c 1000 http://localhost:8000/alloc-segment?biz_tag=test
    美团leaf：ab -n 30000 -c 1000 http://localhost:8080/api/segment/get/leaf-segment-test
    go leaf 
  - 结果：  吞吐率             2845.22    	      1774.78
  	   用户平均等待时间     351.4ms              563.4ms
	   服务器平均等待时间   0.351ms               0.563ms
	   TP99              375ms                  1168ms
    命令2: 
    go-leaf: ab -n 50000 -c 2000 http://localhost:8000/alloc-segment?biz_tag=test
    美团leaf：ab -n 50000 -c 2000 http://localhost:8080/api/segment/get/leaf-segment-test
    go leaf 
  - 结果：  吞吐率             2715.99   	      1739.4
  	   用户平均等待时间     736.38.4ms            1149.8ms
	   服务器平均等待时间   0.368ms               0.575ms
	   TP99              754ms                  11673s

可见，go-leaf 比美团开源Java版 各项数据要领先
	   
 

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
            
        
