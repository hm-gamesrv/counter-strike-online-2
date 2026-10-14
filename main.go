package main

import (
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	. "cso2-server/blademaster/Exp"
	. "cso2-server/blademaster/adminapi"
	. "cso2-server/blademaster/core/achievement"
	. "cso2-server/blademaster/core/automatch"
	. "cso2-server/blademaster/core/chat"
	. "cso2-server/blademaster/core/disassemble"
	. "cso2-server/blademaster/core/holepunch"
	. "cso2-server/blademaster/core/host"
	. "cso2-server/blademaster/core/inventory"
	. "cso2-server/blademaster/core/message"
	. "cso2-server/blademaster/core/notify"
	. "cso2-server/blademaster/core/option"
	. "cso2-server/blademaster/core/playerinfo"
	. "cso2-server/blademaster/core/quick"
	. "cso2-server/blademaster/core/report"
	. "cso2-server/blademaster/core/room"
	. "cso2-server/blademaster/core/shop"
	. "cso2-server/blademaster/core/supply"
	. "cso2-server/blademaster/core/useitem"
	. "cso2-server/blademaster/core/user"
	. "cso2-server/blademaster/core/version"
	. "cso2-server/blademaster/typestruct"
	"cso2-server/builtin"
	. "cso2-server/config"
	. "cso2-server/kerlong"
	. "cso2-server/kerlong/encode"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

var (
	//SERVERVERSION 版本号
	SERVERVERSION = ServerVersion
)

func ReadHead(client net.Conn) ([]byte, bool) {
	SeqBuf := make([]byte, 1)
	headlen := HeaderLen - 1
	head, curlen := make([]byte, headlen), 0
	for {
		n, err := client.Read(SeqBuf)
		if err != nil {
			return head, false
		}
		if n >= 1 && SeqBuf[0] == PacketTypeSignature {
			break
		}
		DebugInfo(2, "Recived a illegal head sig", SeqBuf[0], "from", client.RemoteAddr().String())
	}
	for {
		n, err := client.Read(head[curlen:])

		if err != nil {
			return head, false
		}
		curlen += n
		if curlen >= headlen {
			break
		}
	}
	return head, true
}

func ReadData(client net.Conn, len uint16) ([]byte, bool) {
	data, curlen := make([]byte, len), 0
	for {
		n, err := client.Read(data[curlen:])
		if err != nil {
			return data, false
		}
		curlen += n
		if curlen >= int(len) {
			break
		}
	}
	return data, true
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			// A panic during startup means the server cannot run at all, so the
			// log record is emitted and the process must leave a non-zero exit
			// status: a supervisor (systemd, Docker, Kubernetes) that sees 0
			// would treat a broken configuration as a clean shutdown and
			// neither restart nor alert.
			Error("unhandled panic in main", err)
			os.Exit(1)
		}
	}()

	Info("Counter-Strike Online 2 Server", SERVERVERSION)
	Info("Initializing process ...")

	//get server exe path
	path, err := GetExePath()
	if err != nil {
		panic(err)
	}

	// Configuration, locales and game data are compiled into the binary. The
	// only thing read from next to the executable is an optional "builtin"
	// folder holding overrides.
	builtin.Init(path)

	// Deployment settings come from the environment: ports, locale, data dir.
	// A relative DATA_DIR is resolved against the working directory, which is
	// the usual meaning for an environment variable; the resolved path is
	// logged below either way.
	Env.ReadEnv()

	if err = InitDataDir(Env.DataDir); err != nil {
		panic(err)
	}
	Info("Persistent data directory is", DataDir)
	Info("Environment:", Env.DescribeEnv())

	//read configure
	Conf.InitConf(Env.DataDir)

	InitCSV()
	FullInventoryItem = CreateFullInventoryItem()
	FullInventoryReply = BuildFullInventoryInfo()
	InitCampaignReward()
	if Conf.EnableShop {
		InitShopReply()
	}
	InitBoxReply()

	//read locales
	Locales.InitMotd()
	if Locales.InitLocales() {
		SetLocales()
	}

	//set val
	Level = Conf.DebugLevel
	if Conf.MaxUsers <= 0 {
		MaxUsers = math.MaxUint32
	} else {
		MaxUsers = Conf.MaxUsers
	}

	//init TCP
	server, err := net.Listen("tcp", fmt.Sprintf(":%d", Env.TCPPort))
	if err != nil {
		Error("Cannot listen on TCP port", Env.TCPPort, ":", err)
		panic(err)
	}
	defer server.Close()

	//init UDP
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", Env.UDPPort))
	if err != nil {
		Error("Cannot resolve UDP address:", err)
		panic(err)
	}
	holepunchserver, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		Error("Cannot listen on UDP port", Env.UDPPort, ":", err)
		panic(err)
	}
	defer holepunchserver.Close()

	//Init converter
	InitConverter(Env.CodePage)

	//Init MainServer Info
	MainServer = NewMainServer()
	InitExpTotal()

	//Start UDP Server
	go StartHolePunchServer(strconv.Itoa(int(Env.UDPPort)), holepunchserver)

	//Start TCP Server
	go TCPServer(server)

	//Start BroadCast Service
	go BroadcastRoomList()

	//Start OutdatedItem Service
	go CheckOutdatedItemService()

	//Start SaveUserData Service
	go SaveAllUsersService()

	//Start admin web API
	if Env.EnableAdmin {
		go Start()
	} else {
		Info("Admin web API is disabled (ENABLE_ADMIN=false)")
	}

	// A single SIGINT or SIGTERM shuts the server down: accounts are flushed to
	// disk first, so one deliberate signal is enough.
	//
	// SIGTERM matters even on a development machine: `docker stop` and systemd
	// both send it, and ignoring it would mean the process is killed after a
	// timeout without ever saving.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)
	defer close(ch)

	awaitShutdown(ch)
}

// awaitShutdown blocks until one shutdown signal arrives, flushes every online
// account to disk and returns.
//
// It is a separate function so that the "one signal is enough, and the save
// happens before we exit" behaviour can be tested without signalling the test
// process itself.
func awaitShutdown(ch <-chan os.Signal) {
	sig := <-ch
	Info("Shutdown requested (" + sig.String() + "), saving all online accounts ...")

	if SaveAllUsers() {
		Info("Saved all online users, shutting down")
	} else {
		Error("Failed to save all online users, shutting down anyway")
	}
}

func TCPServer(server net.Listener) {
	defer func() {
		if err := recover(); err != nil {
			Error("TCP server suffered a fault:", err)
		}
	}()

	Info("Server is running at [AnyAdapter]:" + strconv.Itoa(int(Env.TCPPort)))
	for {
		client, err := server.Accept()
		if err != nil {
			DebugInfo(2, "Server accept data error !\n")
			continue
		}
		DebugInfo(2, "Server accept a new connection request at", client.RemoteAddr().String())
		go RecvMessage(client)
	}
}

// RecvMessage 循环处理收到的包
func RecvMessage(client net.Conn) {
	var seq uint8 = 0
	var dataPacket PacketData

	defer client.Close() //关闭con
	defer func() {
		if err := recover(); err != nil {
			OnSendMessage(&seq, client, MessageDialogBox, GAME_SERVER_ERROR)
			Error("Client", client.RemoteAddr().String(), "suffered a fault:", err,
				"dump data", dataPacket.Data, "offset:", dataPacket.CurOffset)
			OnLeaveRoom(client, true)
			DelUserWithConn(client)
		}
	}()

	client.Write([]byte("~SERVERCONNECTED\n"))

	for {
		//读取4字节数据包头部
		headBytes, err := ReadHead(client)
		if !err {
			goto close
		}
		var headPacket PacketHeader
		headPacket.Data = headBytes
		headPacket.PraseHeadPacket()

		//读取数据部分
		bytes, err := ReadData(client, headPacket.Length)
		if !err {
			goto close
		}
		dataPacket = PacketData{Data: bytes, Sequence: headPacket.Sequence, Length: headPacket.Length, Id: bytes[0], CurOffset: 1}

		//执行功能
		switch dataPacket.Id {
		case PacketTypeQuickJoin:
			OnQuick(&dataPacket, client)
		case PacketTypeVersion:
			OnVersionPacket(&seq, client)
		case PacketTypeLogin:
			OnLogin(&seq, &dataPacket, client)
		case PacketTypeRequestChannels:
			OnServerList(client)
		case PacketTypeRequestRoomList:
			OnRoomList(&dataPacket, client)
		case PacketTypeRoom:
			OnRoomRequest(&dataPacket, client)
		case PacketTypeHost:
			OnHost(&dataPacket, client)
		case PacketTypeFavorite:
			OnFavorite(&dataPacket, client)
		case PacketTypeOption:
			OnOption(&dataPacket, client)
		case PacketTypePlayerInfo:
			OnPlayerInfo(&dataPacket, client)
		case PacketTypeChat:
			OnChat(&dataPacket, client)
		case PacketTypeAchievement:
			OnAchievement(&dataPacket, client)
		case PacketTypeAutomatch:
			OnAutoMatch(&dataPacket, client)
		case PacketTypeShop:
			OnShopRequest(&dataPacket, client)
		case PacketTypeReport:
			OnReportRequest(&dataPacket, client)
		case PacketTypeNotify:
			OnNotify(&dataPacket, client)
		case PacketTypeSupply:
			OnSupplyRequest(&dataPacket, client)
		case PacketTypeUseItem:
			OnUseItem(&dataPacket, client)
		case PacketTypeDisassemble:
			OnDisassemble(&dataPacket, client)
		default:
			DebugInfo(2, "Unknown packet", dataPacket.Id, "from", client.RemoteAddr().String())
		}
	}

close:
	DebugInfo(1, "client", client.RemoteAddr().String(), "closed the connection")
	OnLeaveRoom(client, true)
	DelUserWithConn(client)
	return
}

func BroadcastRoomList() {
	for {
		timer := time.NewTimer(6 * time.Second)
		<-timer.C

		for _, v := range UsersManager.SnapshotUsers() {
			if v.CurrentChannelIndex > 0 && v.CurrentRoomId <= 0 {
				OnBroadcastRoomList(v.CurrentChannelServerIndex, v.CurrentChannelIndex, v)
			}
		}

	}
}

func CheckOutdatedItemService() {
	for {
		timer := time.NewTimer(3 * time.Minute)
		<-timer.C

		for _, v := range UsersManager.SnapshotUsers() {
			idxs := v.CheckOutdatedItemIngame()
			DebugInfo(1, "Find", len(idxs), "outdated items for user", v.UserName)
			for _, idx := range idxs {
				rst := BytesCombine(BuildHeader(v.CurrentSequence, PacketTypeInventory_Create),
					BuildInventoryInfoSingle(v, 0, idx))
				SendPacket(rst, v.CurrentConnection)
			}
		}

	}
}

func SaveAllUsersService() {
	for {
		timer := time.NewTimer(1 * time.Minute)
		<-timer.C

		if SaveAllUsers() {
			DebugInfo(1, "Save all users data success !")
		} else {
			DebugInfo(1, "Save all users data failed !")
		}
	}
}
