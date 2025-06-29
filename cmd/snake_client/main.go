package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hustuhao/lockstepserver/pb"
	"github.com/hustuhao/lockstepserver/pkg/packet/pb_packet"
	"github.com/xtaci/kcp-go/v5"
	"google.golang.org/protobuf/proto"
)

var (
	addr = flag.String("addr", "127.0.0.1:10086", "server address")
	room = flag.Uint64("room", 1, "room id")
	id   = flag.Uint64("id", 1, "player id")
)

func main() {
	flag.Parse()

	fmt.Println("addr", *addr, "room", *room, "id", *id)

	ms := &pb_packet.MsgProtocol{}

	c, e := kcp.Dial(*addr)
	if nil != e {
		panic(e)
	}
	defer c.Close()

	// 处理信号，优雅退出
	go handleSignals(c)

	// read
	go func() {
		for {
			n, e := ms.ReadPacket(c)
			if nil != e {
				fmt.Println("read error:", e.Error())
				return
			}

			ret := n.(*pb_packet.Packet)
			msgId := pb.ID(ret.GetMessageID())
			//fmt.Println("receive msg ", msgId.String())
			switch msgId {
			case pb.ID_MSG_Connect:
				msg := &pb.S2C_ConnectMsg{}
				proto.Unmarshal(ret.GetData(), msg)
				if msg.GetErrorCode() != pb.ERRORCODE_ERR_Ok {
					panic(msg.GetErrorCode())
				}
				fmt.Println(msg)
			case pb.ID_MSG_Frame:
				//msg := &pb.S2C_FrameMsg{}
				//proto.Unmarshal(ret.GetData(), msg)
				msg := &pb.S2C_FrameMsg{}
				proto.Unmarshal(ret.GetData(), msg)
				fmt.Println(msg)
				for _, frame := range msg.Frames {
					for _, input := range frame.Input {
						if input.Id != id { // 假设 clientAID 是客户端A的ID
							// 处理客户端B的移动消息，更新本地游戏画面
							updateSnakePosition(*input.Id, *input.Sid, *input.X, *input.Y)
						}
					}
				}
			default:

			}
		}
	}()

	// connect
	if _, e := c.Write(pb_packet.NewPacket(uint8(pb.ID_MSG_Connect), &pb.C2S_ConnectMsg{
		PlayerID: proto.Uint64(*id),
		BattleID: proto.Uint64(*room),
	}).Serialize()); nil != e {
		panic(fmt.Sprintf("write error:%s", e.Error()))
	}
	time.Sleep(time.Second)

	// 心跳,每秒1次
	go sendHeartBeat(c)

	// ready
	if _, e := c.Write(pb_packet.NewPacket(uint8(pb.ID_MSG_JoinRoom), nil).Serialize()); nil != e {
		panic(fmt.Sprintf("write error:%s", e.Error()))
	}
	time.Sleep(time.Second)
	// ready
	if _, e := c.Write(pb_packet.NewPacket(uint8(pb.ID_MSG_Ready), nil).Serialize()); nil != e {
		panic(fmt.Sprintf("write error:%s", e.Error()))
	}
	time.Sleep(time.Second)

	// 监听键盘输入
	go listenKeyboardInput(c)

	// 保持主线程存活
	select {}
}

// 处理信号，优雅退出
func handleSignals(c net.Conn) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received termination signal, closing connection...")
	c.Close()
	os.Exit(0)
}

// 心跳
func sendHeartBeat(c net.Conn) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			// heartbeat
			if _, e := c.Write(pb_packet.NewPacket(uint8(pb.ID_MSG_Heartbeat), nil).Serialize()); nil != e {
				panic(fmt.Sprintf("write error:%s", e.Error()))
			}
		}
	}
}

// 监听键盘输入
func listenKeyboardInput(c net.Conn) {
	// 初始化输入状态
	var inputChan = make(chan rune)
	go func() {
		for {
			var input rune
			fmt.Scanf("%c", &input)
			fmt.Printf("输入:%c\n", input)
			inputChan <- input
		}
	}()

	for {
		select {
		case input := <-inputChan:
			var sid int32
			switch input {
			case 'w':
				sid = 1 // 上
			case 's':
				sid = 2 // 下
			case 'a':
				sid = 3 // 左
			case 'd':
				sid = 4 // 右
			default:
				continue
			}

			x := int32(rand.Intn(1000))
			y := int32(rand.Intn(1000))
			p := pb_packet.NewPacket(uint8(pb.ID_MSG_Input), &pb.C2S_InputMsg{
				Sid: proto.Int32(sid),
				X:   proto.Int32(x),
				Y:   proto.Int32(y),
			})
			fmt.Printf("[SnakeMove] Snake %d moved to position (%d, %d)\n", id, x, y)
			if _, e := c.Write(p.Serialize()); nil != e {
				fmt.Println(fmt.Sprintf("write error:%s", e.Error()))
			}
		}
	}
}

// updateSnakePosition 处理其他客户端的移动消息，更新本地游戏画面
func updateSnakePosition(snakeID uint64, sid int32, x int32, y int32) {
	// 根据接收到的位置信息更新本地游戏逻辑中的蛇的状态
	fmt.Printf("[updateSnakePosition] Snake %d moved to position (%d, %d)\n", snakeID, x, y)

	// 这里可以添加具体的渲染逻辑，例如更新图形界面中对应蛇的位置
	// 示例：gameLogic.UpdateSnakePosition(snakeID, x, y)
}
