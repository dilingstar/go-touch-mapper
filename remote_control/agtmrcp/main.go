package main

import (
	"context" // <-- 修复: 添加 "context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	// "strings" // <-- 修复: 移除 "strings"
	"syscall"
	"time"

	"github.com/kenshaw/evdev"
	"golang.org/x/sys/unix"
)

// --------- GTM 通讯格式定义 (从 main.go.txt 复制) ---------

type dev_type uint8

const (
	type_mouse    = dev_type(0)
	type_keyboard = dev_type(1)
	type_joystick = dev_type(2)
	type_touch    = dev_type(3)
	type_unknown  = dev_type(4)
)

var devTypeFriendlyName = map[dev_type]string{
	type_mouse:    "鼠标",
	type_keyboard: "键盘",
	type_joystick: "手柄",
	type_touch:    "触屏",
	type_unknown:  "未知",
}

// 检查设备类型 (从 main.go.txt 复制 - 已修复)
func check_dev_type(dev *evdev.Evdev) dev_type {
	abs := dev.AbsoluteTypes()
	key := dev.KeyTypes()
	rel := dev.RelativeTypes()
	_, MTPositionX := abs[evdev.AbsoluteMTPositionX]
	_, MTPositionY := abs[evdev.AbsoluteMTPositionY]
	_, MTSlot := abs[evdev.AbsoluteMTPositionX] // 修正: 应该是 evdev.AbsoluteMTSlot
	_, MTTrackingID := abs[evdev.AbsoluteMTTrackingID]
	if MTPositionX && MTPositionY && MTSlot && MTTrackingID {
		return type_touch
	}

	// --- 修复鼠标检测 ---
	_, RelX := rel[evdev.RelativeX]
	_, RelY := rel[evdev.RelativeY]
	_, MouseLeft := key[evdev.BtnLeft]
	// 原始检查 (太严格): if RelX && RelY && HWheel && MouseLeft && MouseRight
	// 很多鼠标没有 HWheel (水平滚轮)
	if RelX && RelY && MouseLeft {
		// 新的、更宽松的检查：只要有X、Y轴和鼠标左键，就认为是鼠标
		return type_mouse
	}
	// --- 修复结束 ---

	keyboard_keys := true
	for i := evdev.KeyEscape; i <= evdev.KeyScrollLock; i++ {
		_, ok := key[i]
		keyboard_keys = keyboard_keys && ok
	}
	if keyboard_keys {
		return type_keyboard
	}

	axis_count := 0
	for i := evdev.AbsoluteX; i <= evdev.AbsoluteRZ; i++ {
		_, ok := abs[i]
		if ok {
			axis_count++
		}
	}
	if axis_count >= 4 {
		return type_joystick
	}
	return type_unknown
}

// 获取设备名 (从 main.go.txt 复制)
func get_dev_name_by_index(index int) string {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		return "读取设备名称失败"
	}
	d := evdev.Open(fd)
	defer d.Close()
	return d.Name()
}

// 获取可用设备 (从 main.go.txt 复制)
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
			// 检查读取权限
			err := unix.Access(fmt.Sprintf("/dev/input/%s", file.Name()), unix.R_OK)
			if err != nil {
				// 没有读取权限，可能是因为没有 root 或 adb
				continue
			}

			fd, err := os.OpenFile(fmt.Sprintf("/dev/input/%s", file.Name()), os.O_RDONLY, 0)
			if err != nil {
				// fmt.Printf("读取设备 /dev/input/%s 失败 : %v \n", file.Name(), err)
				continue
			}
			d := evdev.Open(fd)
			devType := check_dev_type(d)
			d.Close() // 立即关闭，我们只是检查类型

			if devType != type_unknown && devType != type_touch {
				result[index] = devType
			}
		}
	}
	return result
}

// --------- AGTMRCP 核心代码 ---------

var global_close_signal = make(chan bool)

// 我们修改版的设备读取器
func dev_reader_udp(index int, devType dev_type, conn *net.UDPConn, sampleRate time.Duration) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("读取设备 event%d 失败: %v\n", index, err)
		return
	}
	d := evdev.Open(fd)
	dev_name := d.Name()

	// !!! 核心：锁定设备 !!!
	// 这会让手机B的系统忽略这个设备的输入
	err = d.Lock()
	if err != nil {
		fmt.Printf("警告: 锁定设备 %s 失败: %v. 手机B可能会响应输入。\n", dev_name, err)
	} else {
		fmt.Printf("成功锁定设备: %s\n", dev_name)
	}
	defer d.Unlock() // 程序退出时自动解锁
	defer d.Close()

	// 修复: 之前未定义的 context 在这里使用
	event_ch := d.Poll(context.Background())
	events := make([]*evdev.Event, 0)
	var lastSend time.Time // 用于鼠标采样率

	fmt.Printf("开始读取设备: %s\n", dev_name)

	for {
		select {
		case <-global_close_signal:
			fmt.Printf("释放设备: %s\n", dev_name)
			return
		case event := <-event_ch:
			if event == nil {
				fmt.Printf("移除设备: %s\n", dev_name)
				return
			}

			if event.Type == evdev.SyncReport {
				if len(events) == 0 {
					continue // 没有事件，不同步
				}

				// --- 采样率控制逻辑 ---
				if devType == type_mouse && sampleRate > 0 {
					// 检查这个包里是否有鼠标移动 (REL) 事件
					hasRelEvent := false
					for _, e := range events {
						if e.Type == evdev.EventRelative {
							hasRelEvent = true
							break
						}
					}

					if hasRelEvent {
						// 这是一个鼠标移动包
						if time.Since(lastSend) < sampleRate {
							// 时间未到，丢弃这个包 (不发送)
							events = make([]*evdev.Event, 0) // 清空缓冲区
							continue
						}
						// 时间到了，更新发送时间
						lastSend = time.Now()
					}
					// 如果包里只有按键 (KEY) 事件，则立即发送 (不限速)
				}
				// --- 采样率控制结束 ---

				send_udp_packet(conn, events, devType, dev_name)
				events = make([]*evdev.Event, 0) // 清空缓冲区

			} else {
				// 非同步事件，加入缓冲区
				events = append(events, &event.Event)
			}
		}
	}
}

// 封装和发送 UDP 包
func send_udp_packet(conn *net.UDPConn, events []*evdev.Event, devType dev_type, devName string) {
	event_count := len(events)
	if event_count > 255 {
		event_count = 255 // 数量上限
	}

	// 格式: [event_count(1b)] [events(N*8b)] [dev_type(1b)] [dev_name(Nb)]
	buf := make([]byte, 1+event_count*8+1+len(devName))

	buf[0] = byte(event_count)

	offset := 1
	for i := 0; i < event_count; i++ {
		// [Type(2b)] [Code(2b)] [Value(4b)]
		binary.LittleEndian.PutUint16(buf[offset:offset+2], uint16(events[i].Type))
		binary.LittleEndian.PutUint16(buf[offset+2:offset+4], events[i].Code)
		binary.LittleEndian.PutUint32(buf[offset+4:offset+8], uint32(events[i].Value))
		offset += 8
	}

	buf[offset] = byte(devType)
	offset += 1

	copy(buf[offset:], []byte(devName))

	// 发送!
	_, err := conn.Write(buf)
	if err != nil {
		fmt.Printf("! 发送 UDP 包失败: %v\n", err)
	}
}

// 自动检测并启动读取
func auto_detect_and_read_udp(conn *net.UDPConn, sampleRate time.Duration) {
	devices := make(map[int]bool)
	for {
		select {
		case <-global_close_signal:
			return
		default:
			auto_detect_result := get_possible_device_indexes(devices)
			for index, devType := range auto_detect_result {
				devName := get_dev_name_by_index(index)
				// 排除虚拟设备
				if devName == "go-touch-mapper-virtual-device" || devName == "v_touch_screen" {
					continue
				}

				fmt.Printf("检测到设备 %s(/dev/input/event%d) : %s\n", devName, index, devTypeFriendlyName[devType])
				localIndex := index
				localDevType := devType
				go func() {
					devices[localIndex] = true
					dev_reader_udp(localIndex, localDevType, conn, sampleRate)
					devices[localIndex] = false
				}()
			}
			time.Sleep(time.Duration(1) * time.Second) // 每秒检测一次新设备
		}
	}
}

// 主函数
func main() {
	fmt.Println("--- Android GTM 远程客户端 (agtmrcp) ---")

	// 1. 解析参数
	if len(os.Args) < 2 {
		fmt.Println("错误: 缺少目标 IP 和端口。")
		fmt.Println("用法: ./agtmrcp <ip>:<端口> [采样率Hz]")
		fmt.Println("示例 (默认): ./agtmrcp 192.168.1.10:61069")
		fmt.Println("示例 (500Hz): ./agtmrcp 192.168.1.10:61069 500")
		return
	}
	targetAddr := os.Args[1]

	var sampleRate time.Duration = 0
	if len(os.Args) > 2 {
		hz, err := strconv.Atoi(os.Args[2])
		if err != nil || hz <= 0 {
			fmt.Printf("无效的采样率: %s。将使用默认模式。\n", os.Args[2])
		} else {
			// (1 / Hz) * 1_000_000_000 = 纳秒
			sampleRate = time.Duration(1e9/hz) * time.Nanosecond
			fmt.Printf("已设置鼠标采样率上限: %d Hz (间隔: %v)\n", hz, sampleRate)
		}
	} else {
		fmt.Println("使用默认模式 (无鼠标采样率限制)")
	}<

	// 2. 解析 UDP 地址
	udpAddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		fmt.Printf("错误: 解析目标地址 %s 失败: %v\n", targetAddr, err)
		fmt.Println("请检查 IP 和端口是否正确。")
		return
	}

	// 3. 连接 UDP
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Printf("错误: 连接到 %s 失败: %v\n", targetAddr, err)
		fmt.Println("请检查: ")
		fmt.Println("  1. 手机A是否运行了 gtm -r")
		fmt.Println("  2. 两部手机是否在同一个 WiFi 局域网")
		fmt.Println("  3. IP 地址是否为手机A的 IP")
		return
	}
	defer conn.Close()

	fmt.Printf("已连接到 GTM 服务端: %s\n", targetAddr)
	fmt.Println("--- 开始转发 ---")
	fmt.Println("使用热点并贴近设备可有效降低延迟。")
        fmt.Println("开启5Ghz-wifi6热点有效降低鼠标卡顿。")
        fmt.Println("安卓内核询轮率可能被限制在2ms，导致鼠标最高500hz。")
	fmt.Println("按回车键 (Enter) 结束服务。")

	// 4. 启动设备检测
	go auto_detect_and_read_udp(conn, sampleRate)

	// 5. 监听退出
	// 监听 Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 监听回车键
	exitChan := make(chan bool)
	go func() {
		os.Stdin.Read(make([]byte, 1)) // 等待任意输入 (回车)
		exitChan <- true
	}()

	select {
	case <-sigChan:
		fmt.Println("\n接收到终止信号...")
	case <-exitChan:
		fmt.Println("\n接收到回车键...")
	}

	// 6. 优雅退出
	fmt.Println("正在停止服务，释放设备...")
	close(global_close_signal)
	time.Sleep(time.Second * 1) // 等待 dev_reader 协程退出
	fmt.Println("已停止。")
}
