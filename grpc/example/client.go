package main

import (
	"context"
	"fmt"
	pt "go-leaf/grpc/proto"
	"google.golang.org/grpc"
)

func main() {

	conn, err := grpc.Dial(":1234", grpc.WithInsecure())
	if err != nil {
		fmt.Println("连接服务器失败", err)
	}

	defer conn.Close()

	c := pt.NewGreeterClient(conn)

	r1, err := c.Snowflake(context.Background(), &pt.SnowflakeRequest{MachineId: 32})
	r2, err := c.Segment(context.Background(), &pt.SegmentRequest{BizTag: "test"})

	if err != nil {
		fmt.Println("cloud not get Hello server ..", err)
		return
	}

	fmt.Printf("SnowFlake resp: errno: %v  msg: %v id: %v\n", r1.Errno, r1.Mag, r1.Id)
	fmt.Printf("Segment resp: errno: %v  msg: %v id: %v\n", r2.Errno, r2.Mag, r2.Id)
}
