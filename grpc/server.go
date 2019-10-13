package main

import (
	"context"
	"fmt"
	"go-leaf/core"
	hs "go-leaf/grpc/proto"
	pb "go-leaf/grpc/proto"
	"google.golang.org/grpc"
	"log"
	"net"
)

type replay struct {
	Errno int32
	Mag   string
	Id    int64
}
type Server struct{}

func (s *Server) Snowflake(ctx context.Context, in *pb.SnowflakeRequest) (*pb.SnowflakeReply, error) {
	resp := replay{}
	var (
		machineid int32
		err       error
		snow      *core.Snowflake
	)
	if machineid = in.GetMachineId(); machineid == 0 {
		resp.Errno = -1
	}
	//map中有无此机器号
	if exist := core.GAllocSnowflake.BoolmachineMap(machineid); exist {

	} else { //添加到map中
		if err = core.GAllocSnowflake.NewSnowflake(int32(machineid)); err != nil {
			resp.Mag = fmt.Sprintf("%v", err)
		}

	}
	//获取
	snow, _ = core.GAllocSnowflake.MachineMap[machineid]
	if resp.Id, err = snow.NextId(); err != nil {
		return &pb.SnowflakeReply{
			Errno: resp.Errno,
			Mag:   resp.Mag,
			Id:    0,
		}, err
	}

	return &pb.SnowflakeReply{
		Errno: resp.Errno,
		Mag:   "success",
		Id:    resp.Id,
	}, nil

}

func (s *Server) Segment(ctx context.Context, in *pb.SegmentRequest) (*pb.SegmentReply, error) {
	var err error
	resp := replay{}
	if resp.Id, err = core.GAllocSegment.NextId(in.BizTag); err != nil {
		resp.Errno = -1
		resp.Mag = fmt.Sprintf("%v", err)
		resp.Id = 0
	}
	return &pb.SegmentReply{
		Errno: 0,
		Mag:   "success",
		Id:    resp.Id,
	}, err

}

func main() {
	if err := core.Init(); err != nil {
		log.Print(err)
	} else {
		log.Print("Init finshed, Grpc server is starting...")
	}

	grpcServer := grpc.NewServer()
	hs.RegisterGreeterServer(grpcServer, new(Server))

	lis, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(lis)

}
