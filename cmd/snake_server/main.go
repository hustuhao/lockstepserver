// server/snake_server.go
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hustuhao/lockstepserver/cmd/snake_server/api"
	"github.com/hustuhao/lockstepserver/pb/snake"
	"github.com/hustuhao/lockstepserver/pkg/log4gox"
	"github.com/hustuhao/lockstepserver/pkg/packet/pb_packet"
	"github.com/hustuhao/lockstepserver/server"

	l4g "github.com/alecthomas/log4go"
)

// go run cmd/snake_server/main.go
// sh cmd/snake_server/create_room.sh
// go run cmd/snake_client/main.go -room=1 -id=1
// go run cmd/snake_client/main.go -room=1 -id=2

var (
	httpAddress = flag.String("web", ":80", "web listen address")
	udpAddress  = flag.String("udp", ":10086", "udp listen address(':10086' means localhost:10086)")
	debugLog    = flag.Bool("log", true, "debug log")
)

func main() {
	flag.Parse()

	l4g.Close()
	l4g.AddFilter("debug logger", l4g.DEBUG, log4gox.NewColorConsoleLogWriter())

	s, err := server.New(*udpAddress)
	if err != nil {
		panic(err)
	}
	_ = api.NewWebAPI(*httpAddress, s.RoomManager())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)
	ticker := time.NewTimer(time.Minute)
	defer ticker.Stop()

	l4g.Info("[main] start...")
	// 主循环
QUIT:
	for {
		select {
		case sig := <-sigs:
			l4g.Info("Signal: %s", sig.String())
			break QUIT
		case <-ticker.C:
			// todo
			fmt.Println("room number ", s.RoomManager().RoomNum())
		}
	}
	l4g.Info("[main] quiting...")
	s.Stop()
}

// 处理贪吃蛇操作输入消息
func processSnakeInput(s *server.LockStepServer, playerID uint64, msg *pb_packet.Packet) {
	m := &snake.C2S_SnakeInputMsg{}
	if err := msg.Unmarshal(m); err != nil {
		l4g.Error("[processSnakeInput] Unmarshal error: %v", err)
		return
	}

	// Update snake direction for the player
	room := s.RoomManager().GetRoom(playerID)
	if room == nil {
		l4g.Error("[processSnakeInput] Player %d not in any room", playerID)
		return
	}
	if room.HasPlayer(playerID) {
		// Create a new snake state for this player
		snakeState := &snake.SnakeState{
			Id:        playerID,
			Direction: m.Direction,
			// Initialize positions (this should be updated based on actual game logic)
			Positions: []int32{0, 0},
		}

		// Broadcast this change to all players
		broadcastSnakeFrame(s, 0, []*snake.SnakeState{snakeState}, nil)
	}

	l4g.Info("[processSnakeInput] Player %d changed direction to %s", playerID, m.Direction.String())
}

// 广播贪吃蛇帧数据
func broadcastSnakeFrame(s *server.LockStepServer, frameID uint32, snakes []*snake.SnakeState, foods []*snake.FoodState) {
	msg := &snake.S2C_SnakeFrameMsg{
		FrameId: frameID,
		Snakes:  snakes,
		Foods:   foods,
	}

	packet := pb_packet.NewPacket(uint8(snake.ID_ID_MSG_SnakeFrame), msg)
	if packet == nil {
		l4g.Error("[broadcastSnakeFrame] Failed to create packet")
		return
	}
	// 广播消息给所有玩家
	//s.RoomManager().Broadcast(packet)
}
