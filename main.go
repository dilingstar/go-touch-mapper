package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/akamensky/argparse"
	"github.com/kenshaw/evdev"
	"go.bug.st/serial"
)

// --- [修改 V2.0.0] 1. 设置版本号 ---
var go_build_version string = "外星分支v2.0.0"

// --- [修改 V2.0.0] 1. 结束 ---
var uinput_keyboard_mouse_dev_name = ""

type event_pack struct {
	//表示一个动作 由一系列event组成
	dev_name string
	dev_type dev_type
	events   []*evdev.Event
}

type touch_control_pack struct {
	//触屏控制信息
	action   int8
	id       int32
	x        int32
	y        int32
	screen_x int32
	screen_y int32
}

type u_input_control_pack struct {
	action int8
	arg1   int32
	arg2   int32
}

type touch_control_func func(data touch_control_pack)

func dev_reader(event_reader chan *event_pack, index int) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		logger.Errorf("读取设备失败 : %v", err)
		return
	}
	d := evdev.Open(fd)
	defer d.Close()
	event_ch := d.Poll(context.Background())
	events := make([]*evdev.Event, 0)
	dev_name := d.Name()
	logger.Infof("开始读取设备 : %s", dev_name)
	d.Lock()
	defer d.Unlock()
	for {
		select {
		case <-global_close_signal:
			logger.Infof("释放设备 : %s", dev_name)
			return
		case event := <-event_ch:
			if event == nil {
				logger.Warnf("移除设备 : %s", dev_name)
				return
			} else if event.Type == evdev.SyncReport {
				pack := &event_pack{
					dev_name: dev_name,
					dev_type: check_dev_type(d),
					events:   events,
				}
				event_reader <- pack
				events = make([]*evdev.Event, 0)
			} else {
				events = append(events, &event.Event)
			}
		}
	}
}

func touch_dev_reader(event_reader chan *event_pack, index int) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		logger.Errorf("读取设备失败 : %v", err)
		return
	}
	d := evdev.Open(fd)
	defer d.Close()
	event_ch := d.Poll(context.Background())
	events := make([]*evdev.Event, 0)
	dev_name := d.Name()
	abs_max_x := d.AbsoluteTypes()[evdev.AbsoluteMTPositionX].Max
	abs_max_y := d.AbsoluteTypes()[evdev.AbsoluteMTPositionY].Max

	logger.Infof("开始读取设备 : %s", dev_name)
	d.Lock()
	defer d.Unlock()
	for {
		select {
		case <-global_close_signal:
			logger.Infof("释放设备 : %s", dev_name)
			return
		case event := <-event_ch:
			if event == nil {
				logger.Warnf("移除设备 : %s", dev_name)
				return
			} else if event.Type == evdev.SyncReport {
				pack := &event_pack{
					dev_name: dev_name,
					dev_type: check_dev_type(d),
					events:   events,
				}
				event_reader <- pack
				events = make([]*evdev.Event, 0)
			} else {
				if event.Event.Code == absMtPositionX {
					event.Event.Value = int32(int64(event.Event.Value) * 0x7ffffffe / int64(abs_max_x))
				}
				if event.Event.Code == absMtPositionY {
					event.Event.Value = int32(int64(event.Event.Value) * 0x7ffffffe / int64(abs_max_y))
				}
				events = append(events, &event.Event)
			}
		}
	}
}

func udp_event_injector(ch chan *event_pack, port int) {
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	})
	if err != nil {
		logger.Errorf("udp error : %v", err)
		return
	}
	defer listen.Close()

	recv_ch := make(chan []byte)
	go func() {
		for {
			var buf [1024]byte
			n, _, err := listen.ReadFromUDP(buf[:])
			if err != nil {
				break
			}
			recv_ch <- buf[:n]
		}
	}()
	logger.Infof("已准备从以下地址接收远程事件: ")
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue // 跳过非 IPv4 地址
			}
			if !ipv4.IsLoopback() {
				logger.Infof("UDP://%s:%v", ipv4, port)
			}
		}
	}
	for {
		select {
		case <-global_close_signal:
			return
		case pack := <-recv_ch: //数据包格式：<event_count:1byte><event1:8byte><event2:8byte>...<eventN:8byte><dev_type:1byte><dev_name:N byte>
			//每个event格式：<type:2byte><code:2byte><value:4byte>
			//共8byte
			// logger.Debugf("%v", pack)
			event_count := int(pack[0])
			events := make([]*evdev.Event, 0)
			for i := 0; i < event_count; i++ {
				event := &evdev.Event{
					Type:  evdev.EventType(uint16(binary.LittleEndian.Uint16(pack[8*i+1 : 8*i+3]))),
					Code:  uint16(binary.LittleEndian.Uint16(pack[8*i+3 : 8*i+5])),
					Value: int32(binary.LittleEndian.Uint32(pack[8*i+5 : 8*i+9])),
				}
				// logger.Debugf("%v", event)
				events = append(events, event)
			}
			e_pack := &event_pack{
				dev_name: string(pack[event_count*8+2:]),
				dev_type: dev_type(pack[event_count*8+1]),
				events:   events,
			}
			ch <- e_pack
			// logger.Debugf("接收到事件 : %v", e_pack)
		}
	}
}

var global_close_signal = make(chan bool)  //仅会在程序退出时关闭  不用于其他用途
var global_is_wordking_remote bool = false // 是否正在远程控制 并且无法获取触屏状态
var global_device_orientation int32 = 0
var global_screen_x int32 = 1000
var global_screen_y int32 = 1000

func get_device_orientation() int32 {
	output, err := exec.Command("sh", "-c", "dumpsys input").Output()
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`orientation=(\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		orientation := matches[1]
		result, err := strconv.Atoi(string(orientation))
		if err != nil {
			return 0
		} else {
			return int32(result)
		}
	} else {
		return 0
	}
}

func listen_device_orientation() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			var now_orientation int32 = get_device_orientation()
			if global_device_orientation != now_orientation {
				global_device_orientation = now_orientation
				logger.Debugf("设备方向改变\t[%d]", now_orientation)
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

func rotateAbsoluteXY(x, y int32) (int32, int32) { //根据方向旋转坐标
	switch global_device_orientation {
	case 0:
		return x, y
	case 1:
		return 0x7ffffffe - y, x
	case 2:
		return 0x7ffffffe - x, 0x7ffffffe - y
	case 3:
		return y, 0x7ffffffe - x
	default:
		return x, y
	}
}

type dev_type uint8

const (
	type_mouse    = dev_type(0)
	type_keyboard = dev_type(1)
	type_joystick = dev_type(2)
	type_touch    = dev_type(3)
	type_unknown  = dev_type(4)
)

func check_dev_type(dev *evdev.Evdev) dev_type {
	abs := dev.AbsoluteTypes()
	key := dev.KeyTypes()
	rel := dev.RelativeTypes()
	_, MTPositionX := abs[evdev.AbsoluteMTPositionX]
	_, MTPositionY := abs[evdev.AbsoluteMTPositionY]
	_, MTSlot := abs[evdev.AbsoluteMTSlot]
	_, MTTrackingID := abs[evdev.AbsoluteMTTrackingID]
	if MTPositionX && MTPositionY && MTSlot && MTTrackingID {
		return type_touch //触屏检测这几个abs类型即可
	}

	// --- [修改 V2.0.0] 4. 移植分支的鼠标检测逻辑 ---
	_, RelX := rel[evdev.RelativeX]
	_, RelY := rel[evdev.RelativeY]
	// --- [修改 V1.0.0] 按照您的要求，只检测 X, Y, 和左键 ---
	_, MouseLeft := key[evdev.BtnLeft]
	// _, MouseRight := key[evdev.BtnRight]
	// _, HWheel := rel[evdev.RelativeHWheel] // 移除对横向滚轮的强制要求
	if RelX && RelY && MouseLeft {
		// if RelX && RelY && HWheel && MouseLeft && MouseRight { // 旧的严格检测
		// --- [修改 V1.0.0] 修改完毕 ---
		return type_mouse //鼠标 检测XY 滚轮 左右中键
	}
	// --- [修改 V2.0.0] 4. 结束 ---

	keyboard_keys := true
	for i := evdev.KeyEscape; i <= evdev.KeyScrollLock; i++ {
		_, ok := key[i]
		keyboard_keys = keyboard_keys && ok
	}
	if keyboard_keys {
		return type_keyboard //键盘 检测keycode(1-70)
	}

	axis_count := 0
	for i := evdev.AbsoluteX; i <= evdev.AbsoluteRZ; i++ {
		_, ok := abs[i]
		if ok {
			axis_count++
		}
	}
	if axis_count >= 4 { //检测有两个摇杆
		return type_joystick
	}
	return type_unknown
}

func get_possible_device_indexes(skipList map[int]bool) map[int]dev_type {
	files, _ := ioutil.ReadDir("/dev/input")
	result := make(map[int]dev_type)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if len(file.Name()) <= 5 {
			continue
		}
		if file.Name()[:5] != "event" {
			continue
		}
		index, _ := strconv.Atoi(file.Name()[5:])
		reading, exist := skipList[index]
		if exist && reading {
			continue
		} else {
			fd, err := os.OpenFile(fmt.Sprintf("/dev/input/%s", file.Name()), os.O_RDONLY, 0)
			if err != nil {
				logger.Errorf("读取设备/dev/input/%s失败 : %v ", file.Name(), err)
			}
			d := evdev.Open(fd)
			defer d.Close()
			devType := check_dev_type(d)
			if devType != type_unknown {
				result[index] = devType
			}
		}
	}
	return result
}

func get_dev_name_by_index(index int) string {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		return "读取设备名称失败"
	}
	d := evdev.Open(fd)
	defer d.Close()
	return d.Name()
}

func execute_view_move(handelerInstance *TouchHandler, x, stepValue, sleepMS int) {
	handelerInstance.handel_view_move(0, 0)
	time.Sleep(time.Millisecond * time.Duration(sleepMS))
	if x > 0 {
		steps := x / stepValue
		for i := 0; i < steps; i++ {
			if !handelerInstance.map_on {
				break
			}
			handelerInstance.handel_view_move(int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(int32(x%stepValue), 0)
	} else {
		steps := -x / stepValue
		for i := 0; i < steps; i++ {
			if !handelerInstance.map_on {
				break
			}
			handelerInstance.handel_view_move(-int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(-int32(x%stepValue), 0)
	}
}

func stdin_control_view_move(handelerInstance *TouchHandler) {
	logger.Info("输入数值以精确控制view移动 [ x | x 步长 间隔 ]")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		var x, stepValue, sleepMS int
		var err error
		if len(args) == 1 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			} else {
				stepValue = 24
				sleepMS = 16
			}

		} else if len(args) == 3 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			stepValue, err = strconv.Atoi(args[1])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			sleepMS, err = strconv.Atoi(args[2])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
		} else {
			logger.Error("参数错误 usage:x | x stepValue sleepMS")
			continue
		}
		logger.Infof("x: %d stepValue: %d sleepMS: %d", x, stepValue, sleepMS)
		if handelerInstance.map_on {
			execute_view_move(handelerInstance, x, stepValue, sleepMS)
		} else {
			logger.Info("等待映射开关打开中...")
			for {
				if !handelerInstance.map_on {
					time.Sleep(time.Duration(100) * time.Millisecond)
				} else {
					execute_view_move(handelerInstance, x, stepValue, sleepMS)
					break
				}
			}
		}
	}
}

func get_MT_size(indexes map[int]bool) (int32, int32) { //获取MTPositionX和MTPositionY的max值
	//在1+7p上是等于get_wm_size
	//但是红魔7sp应该是为了更精确，使用了缩放,实际的数值为 rawValue >> 3
	// rawValue * wm_size / mt_size = true_value
	for index, _ := range indexes {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			logger.Errorf("get_MT_size error:%v", err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		abs := d.AbsoluteTypes()
		MTPositionX, _ := abs[evdev.AbsoluteMTPositionX]
		MTPositionY, _ := abs[evdev.AbsoluteMTPositionY]
		return MTPositionX.Max, MTPositionY.Max
	}
	return int32(1), int32(1)
}

func OpenSerialWritePipe(portName string, baudRate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}
	return port, nil
}

func auto_detect_and_read(event_chan chan *event_pack, patern string) {
	//自动检测设备并读取 循环检测 自动管理设备插入移除
	devices := make(map[int]bool)
	for {
		select {
		case <-global_close_signal:
			return
		default:
			auto_detect_result := get_possible_device_indexes(devices)
			devTypeFriendlyName := map[dev_type]string{
				type_mouse:    "鼠标",
				type_keyboard: "键盘",
				type_joystick: "手柄",
				type_touch:    "触屏",
				type_unknown:  "未知",
			}
			for index, devType := range auto_detect_result {
				devName := get_dev_name_by_index(index)
				if devName == uinput_keyboard_mouse_dev_name {
					continue //跳过生成的虚拟设备
				}
				re := regexp.MustCompile(patern)
				if !re.MatchString(devName) {
					// logger.Debugf("设备名称 %s 不匹配 %s", devName, patern)
					continue
				}
				if devType == type_mouse || devType == type_keyboard || devType == type_joystick {
					logger.Infof("检测到设备 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
					localIndex := index
					go func() {
						devices[localIndex] = true
						dev_reader(event_chan, localIndex)
						devices[localIndex] = false
					}()
				}
			}
			time.Sleep(time.Duration(400) * time.Millisecond)
		}
	}
}

func parseHIDTouchScreen(s string) (x, y uint32, r int, err error) {
	parts := strings.Split(s, "x")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid format: expected 3 parts separated by 'x', got %d", len(parts))
	}
	x64, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid x value: %v", err)
	}
	x = uint32(x64)
	y64, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid y value: %v", err)
	}
	y = uint32(y64)
	r, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid r value: %v", err)
	}
	if r < 0 || r > 3 {
		return 0, 0, 0, fmt.Errorf("r value must be 0, 1, 2, or 3, got %d", r)
	}
	return x, y, r, nil
}

func parseSenderAddress(s string, defaultPort int) (string, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		ip := parts[0]
		return ip, defaultPort, nil
	}
	if len(parts) == 2 {
		ip := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid port value: %v", err)
		}
		return ip, port, nil
	}
	return "", 0, fmt.Errorf("invalid address format: expected 'IP' or 'IP:PORT', got %s", s)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

//go:embed configs
var configs embed.FS

func main() {

	parser := argparse.NewParser("go-touch-mapper", " ")

	var configPath *string = parser.String("c", "config", &argparse.Options{
		Required: false,
		Help:     "指定配置文件路径，如果不存在则会使用默认模板创建",
	})

	var create_js_info *bool = parser.Flag("", "create-js-info", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "创建手柄配置文件模式",
	})

	var patern *string = parser.String("", "pattern", &argparse.Options{
		Required: false,
		Default:  ".*",
		Help:     "用于筛选设备名称的正则",
	})

	var as_remote_control *string = parser.String("s", "sender", &argparse.Options{
		Required: false,
		Default:  "",
		Help:     "发送本地事件到远程，输入 IP:PORT 例如 192.168.3.7:61069 或者仅输入IP使用默认端口61069",
	})

	var control_mode *string = parser.String("m", "mode", &argparse.Options{
		Required: false,
		Default:  "uinput",
		Help:     "触摸方案，可用控制模式:    \tuinput:\t\t使用uinput创建虚拟触屏  \tinputmanager:\t通过UDS控制安卓inputManager \thid:\t\t通过串口控制单片机模拟usb触屏  \totg:\t\t本机配置LinuxUSBgadget模拟usb触屏,设备文件为/dev/hidg0 \tdirect:\t\t直接写入设备真实触屏,需要root权限或者低版本安卓",
	})

	var mixTouchDisabled *bool = parser.Flag("t", "disable-mix", &argparse.Options{
		Required: false,
		Help:     "关闭触屏混合,仅在uinput与inputmanager模式生效",
		Default:  false,
	})

	var uinputMouseKeyboardDisabled *bool = parser.Flag("u", "disable-uinput", &argparse.Options{
		Required: false,
		Help:     "不再创建uinput鼠标键盘设备,仅在uinput、inputmanager与direct模式生效",
		Default:  false,
	})

	var usingInputManagerDisplayID *int = parser.Int("", "display-id", &argparse.Options{
		Required: false,
		Default:  0,
		Help:     "显示器ID,仅inputmanager模式生效,多显示器情况下可控制额外的显示器",
	})

	var usingHIDTouchTtyPath *string = parser.String("", "tty-path", &argparse.Options{
		Required: false,
		Default:  "",
		Help:     "串口设备路径,hid模式下必须指定",
	})

	var usingDeviceRotation *int = parser.Int("", "rotation", &argparse.Options{
		Required: false,
		Default:  1,
		Help:     "手动指定屏幕方向,仅在hid与otg模式下需要(建议使用-v)",
	})

	var using_remote_control *bool = parser.Flag("r", "remote-control", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "是否从UDP接收远程事件",
	})

	var port *int = parser.Int("p", "port", &argparse.Options{
		Required: false,
		Help:     "指定监听远程事件的UDP端口号与控制后台端口",
		Default:  61069,
	})

	var using_v_mouse *bool = parser.Flag("v", "v-mouse", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "用触摸操作模拟鼠标,需要额光标外显示程序vPointer,同时可用于在hid与otg模式下获取屏幕方向",
	})

	var using_remote_v_mouse *string = parser.String("", "v-mouse-addr", &argparse.Options{
		Required: false,
		Default:  "",
		Help:     "模拟光标显示程序地址,默认本机6533端口,输入IP:PORT,例如192.168.3.7:6533,或者仅输入IP使用默认端口6533",
	})

	// --- [修改 V2.0.0] 2. 移除 --auto-release 参数 ---
	// var view_release_timeout *int = parser.Int("", "auto-release", &argparse.Options{
	// 	Required: false,
	// 	Help:     "触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用",
	// 	Default:  200,
	// })
	// --- [修改 V2.0.0] 2. 结束 ---

	var measure_sensitivity_mode *bool = parser.Flag("", "measure-mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调,用于测试灵敏度",
	})

	var debug_mode *bool = parser.Flag("d", "debug-mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "打印debug信息",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if go_build_version == "" {
		go_build_version = "DEV"
	}
	uinput_keyboard_mouse_dev_name = fmt.Sprintf("EVO80_Keyboard_%s", go_build_version)
	logger.Infof("当前构建版本: %v", go_build_version)

	if *debug_mode {
		logger.WithDebug()
		logger.Debug("debug on")
	}

	if *create_js_info {
		//=================================================================================================================================
		// 创建手柄配置文件部分
		auto_detect_result := get_possible_device_indexes(make(map[int]bool))
		devTypeFriendlyName := map[dev_type]string{
			type_mouse:    "鼠标",
			type_keyboard: "键盘",
			type_joystick: "手柄",
			type_touch:    "触屏",
			type_unknown:  "未知",
		}
		for index, devType := range auto_detect_result {
			devName := get_dev_name_by_index(index)
			logger.Infof("检测到设备 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
		}
		js_events := make([]int, 0)
		for index, devType := range auto_detect_result {
			if devType == type_joystick {
				js_events = append(js_events, index)
			}
		}
		if len(js_events) == 1 {
			create_js_info_file(js_events[0])
		} else {
			if len(js_events) == 0 {
				logger.Warn("未检测到手柄")
			} else {
				logger.Warn("检测到多个手柄,断开其他手柄的连接")
			}
		}
		return
		//=================================================================================================================================
	} else if *as_remote_control != "" {
		//=================================================================================================================================
		// 远程事件发送器部分
		ip, port, err := parseSenderAddress(*as_remote_control, 61069)
		if err != nil {
			logger.Errorf("解析发送地址失败: %v", err)
			return
		}
		logger.Infof("启动远程事件发送器 目标地址 %s:%d", ip, port)
		events_ch := make(chan *event_pack) //主要设备事件管道
		go auto_detect_and_read(events_ch, *patern)
		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
		if err != nil {
			fmt.Println("Dial UDP failed, err:", err)
			return
		}
		defer conn.Close()

		pack_count := 0
		pack_size := 0
		go (func() {
			ticker := time.NewTicker(time.Second * 1)
			for {
				select {
				case <-global_close_signal:
					return
				case <-ticker.C:
					logger.Debugf("发送频率: %d Pack/s 发送数据量: %d B/s\r", pack_count, pack_size)
					pack_count = 0
					pack_size = 0
				}
			}
		})()

		data := make([]byte, 4096)
		for {
			select {
			case <-global_close_signal:
				return
			case pack := <-events_ch:
				event_count := len(pack.events)
				data[0] = byte(event_count)
				for i, event := range pack.events {
					offset := 1 + i*8
					binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(event.Type))
					binary.LittleEndian.PutUint16(data[offset+2:offset+4], event.Code)
					binary.LittleEndian.PutUint32(data[offset+4:offset+8], uint32(event.Value))
				}
				name_offset := 1 + event_count*8
				data[name_offset] = byte(pack.dev_type)
				if pack.dev_type == type_keyboard || pack.dev_type == type_mouse {
					pack.dev_name = "rkm"
				}
				copy(data[name_offset+1:], []byte(pack.dev_name))
				data_length := name_offset + 1 + len(pack.dev_name)
				_, err = conn.Write(data[:data_length])
				pack_count++
				pack_size += data_length
				if err != nil {
					fmt.Println("Send data failed, err:", err)
					return
				}
			}
		}
		//=================================================================================================================================
	} else {
		switch *control_mode {
		case "uinput":
		case "inputmanager":
		case "hid":
			if *usingHIDTouchTtyPath == "" {
				logger.Error("使用hid模式需要使用--tty-path参数指定串口设备路径")
				os.Exit(1)
			}
		case "otg":
		case "direct":
		default:
			logger.Errorf("未知模式%s,可用模式:uinput,inputmanager,hid,otg,direct", *control_mode)
			os.Exit(1)
		}

		if *configPath == "" {
			logger.Warn("未指定配置文件，使用默认配置文件")
			exePath, _ := os.Executable()
			exeDir := filepath.Dir(exePath)
			*configPath = filepath.Join(exeDir, "default.json")
		}
		if !fileExists(*configPath) {
			bytes, _ := configs.ReadFile("configs/EXAMPLE.JSON")
			logger.Infof("已从模板创建配置文件:%v", *configPath)
			os.WriteFile(*configPath, bytes, 0644)
		}
		logger.Infof("使用配置文件: %v", *configPath)
		//=================================================================================================================================
		main_events_ch := make(chan *event_pack)                       //主要设备事件管道
		mix_touch_event_ch := make(chan *event_pack)                   //触屏设备事件管道
		u_input_control_ch := make(chan *u_input_control_pack)         //uinput控制键鼠的事件管道
		fileted_u_input_control_ch := make(chan *u_input_control_pack) //v-mouse下过滤拦截事件后的管道,如果不使用vmouse 则fileted_u_input_control_ch直接连接到u_input_control_ch

		go auto_detect_and_read(main_events_ch, *patern)

		if !*mixTouchDisabled && (*control_mode == "uinput" || *control_mode == "inputmanager") {
			for index, devType := range get_possible_device_indexes(make(map[int]bool)) {
				if devType == type_touch {
					logger.Infof("启用触屏混合 %s(/dev/input/event%d)", get_dev_name_by_index(index), index)
					go touch_dev_reader(mix_touch_event_ch, index)
				}
			}
		}

		var touch_control_func touch_control_func

		switch *control_mode {
		case "uinput":
			logger.Info("触屏控制将使用uinput在本机处理")
			if *uinputMouseKeyboardDisabled {
				go (func() {
					for {
						select {
						case <-global_close_signal:
							return
						case <-fileted_u_input_control_ch:
						}
					}
				})()
			} else {
				go handel_u_input_mouse_keyboard(fileted_u_input_control_ch)
			}
			touch_control_func = handel_touch_using_uinput_touch()
		case "inputmanager":
			logger.Info("触屏控制将使用inputManager在本机处理")
			if *uinputMouseKeyboardDisabled {
				go (func() {
					for {
						select {
						case <-global_close_signal:
							return
						case <-fileted_u_input_control_ch:
						}
					}
				})()
			} else {
				go handel_u_input_mouse_keyboard(fileted_u_input_control_ch)
			}
			touch_control_func = handel_touch_using_input_manager(*usingInputManagerDisplayID)
		case "hid":
			global_is_wordking_remote = true
			if *usingDeviceRotation < 0 || *usingDeviceRotation > 3 {
				logger.Error("旋转参数错误 可用选值有 0(竖屏) 1(横屏) 2(反向竖屏) 3(反向横屏)")
				return
			}
			logger.Info("触屏控制将使用串口控制外接的HID设备发送至主机")
			logger.Infof("串口路径：%s", *usingHIDTouchTtyPath)
			logger.Infof("触屏方向：%d", *usingDeviceRotation)
			port, err := OpenSerialWritePipe(*usingHIDTouchTtyPath, 2000000)
			if err != nil {
				logger.Errorf("无法打开串口: %v", err)
				os.Exit(1)
			}
			go (func() {
				for {
					select {
					case <-global_close_signal:
						return
					case <-fileted_u_input_control_ch:
					}
				}
			})()
			touch_control_func = handel_touch_using_hid_manager(port)
		case "otg":
			global_is_wordking_remote = true
			logger.Info("触屏控制将使用本机模拟为HID设备发送至主机")
			logger.Infof("触屏方向：%d", *usingDeviceRotation)
			global_device_orientation = int32(*usingDeviceRotation)
			touch_control_func = handel_touch_using_otg_manager()
			go (func() {
				for {
					select {
					case <-global_close_signal:
						return
					case <-fileted_u_input_control_ch:
					}
				}
			})()
		case "direct":
			if *uinputMouseKeyboardDisabled {
				go (func() {
					for {
						select {
						case <-global_close_signal:
							return
						case <-fileted_u_input_control_ch:
						}
					}
				})()
			} else {
				go handel_u_input_mouse_keyboard(fileted_u_input_control_ch)
			}
			logger.Info("触屏控制将使用直接写入真实设备文件")
			var direct_touch_index int
			for index, devType := range get_possible_device_indexes(make(map[int]bool)) {
				if devType == type_touch {
					logger.Infof("将会直接写入触屏 %s(/dev/input/event%d)", get_dev_name_by_index(index), index)
					direct_touch_index = index
					break
				}
			}
			touch_control_func = handel_touch_using_direct_touch(direct_touch_index)
		default:
			logger.Errorf("未知模式%s,可用模式:uinput,inputmanager,hid,otg,direct", *control_mode)
			os.Exit(1)
		}

		map_switch_signal := make(chan bool) //通知虚拟鼠标当前为鼠标还是映射模式
		touchHandler := InitTouchHandler(
			*configPath,
			main_events_ch,
			touch_control_func,
			u_input_control_ch,
			map_switch_signal,
			*measure_sensitivity_mode,
		)
		if !global_is_wordking_remote { //只有本机运行的时候 才有必要开启触屏混合
			go touchHandler.mix_touch(mix_touch_event_ch)
			if !*using_v_mouse {
				go listen_device_orientation()
			}
		}

		// --- [修改 V2.0.0] 3. 适配 auto_handel_view_release (移除参数) ---
		go touchHandler.auto_handel_view_release()
		// --- [修改 V2.0.0] 3. 结束 ---

		go touchHandler.loop_handel_wasd_wheel()
		go touchHandler.loop_handel_rs_move()

		// --- [修改 V2.0.0] 5. 移植分支的独有 goroutine ---
		go touchHandler.loop_handel_wheel_planet()
		go touchHandler.loop_auto_release_scroll_slider()
		// --- [修改 V2.0.0] 5. 结束 ---

		go touchHandler.handel_event()

		if *using_v_mouse {
			ip := net.IPv4(0, 0, 0, 0)
			port := 6533
			if *using_remote_v_mouse != "" {
				s_ip, s_port, err := parseSenderAddress(*using_remote_v_mouse, 6533)
				if err != nil {
					logger.Errorf("解析模拟光标显示程序地址失败: %v", err)
					return
				}
				ip = net.ParseIP(s_ip)
				port = s_port
			}
			v_mouse := init_v_mouse_controller(touchHandler, u_input_control_ch, fileted_u_input_control_ch, map_switch_signal, net.UDPAddr{
				IP:   ip,
				Port: port,
			})
			go v_mouse.main_loop()
			go v_mouse.loop_handel_v_mouse_wheel_move()
		} else {
			go (func() {
				for {
					select {
					case tmp := <-u_input_control_ch:
						fileted_u_input_control_ch <- tmp
					case <-map_switch_signal:
					}
				}
			})()
		}

		if *using_remote_control {
			logger.Errorf("使用远程控制中。。。。")
			go udp_event_injector(main_events_ch, *port)
		}

		if *measure_sensitivity_mode {
			go stdin_control_view_move(touchHandler)
		}
		go serve(*port, *configPath, touchHandler.reloadConfigure) //启动服务器
		exitChan := make(chan os.Signal, 1)
		signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
		<-exitChan
		close(global_close_signal)
		logger.Info("已停止")
		time.Sleep(time.Millisecond * 40)
	}
}