package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/kenshaw/evdev"
)

type TouchHandler struct {
	events                  chan *event_pack            //接收事件的channel
	touch_control_func      touch_control_func          //发送触屏控制信号的channel
	u_input                 chan *u_input_control_pack  //发送u_input控制信号的channel
	map_on                  bool                        //映射模式开关
	view_id                 int32                       //视角的触摸ID
	wheel_id                int32                       //左摇杆的触摸ID
	allocated_id            []bool                      //10个触摸点分配情况
	config                  *simplejson.Json            //映射配置文件
	joystickInfo            map[string]*simplejson.Json //所有摇杆配置文件 dev_name 为key
	screen_x                int32                       //屏幕宽度
	screen_y                int32                       //屏幕高度
	rel_screen_x            int32
	rel_screen_y            int32
	view_init_x             int32 //初始化视角映射的x坐标
	view_init_y             int32 //初始化视角映射的y坐标
	view_current_x          int32 //当前视角映射的x坐标
	view_current_y          int32 //当前视角映射的y坐标
	view_speed_x            int32 //视角x方向的速度
	view_speed_y            int32 //视角y方向的速度
	rs_speed_x              float64
	rs_speed_y              float64
	wheel_init_x            int32 //初始化左摇杆映射的x坐标
	wheel_init_y            int32 //初始化左摇杆映射的y坐标
	wheel_range             int32 //左摇杆的x轴范围
	wheel_wasd              []string
	view_lock               sync.Mutex //视角控制相关的锁 用于自动释放和控制相关
	wheel_lock              sync.Mutex //左摇杆控制相关的锁 用于自动释放和控制相关
	touch_control_lock      sync.Mutex
	auto_release_view_count int32    //自动释放计时器 有视角移动则重置 否则100ms加一 超过1s 自动释放
	abs_last                sync.Map //abs值的上一次值 用于手柄
	using_joystick_name     string   //当前正在使用的手柄 针对不同手柄死区不同 但程序支持同时插入多个手柄 因此会识别最进发送事件的手柄作为死区配置
	ls_wheel_released       bool     //左摇杆滚轮释放
	wasd_wheel_released     bool     //wasd滚轮释放 两个都释放时 轮盘才会释放
	wasd_wheel_last_x       int32    //wasd滚轮上一次的x坐标
	wasd_wheel_last_y       int32    //wasd滚轮上一次的y坐标
	wasd_up_down_statues    []bool
	key_action_state_save   sync.Map
	BTN_SELECT_UP_DOWN      int32
	// KEYBOARD_SWITCH_KEY_NAME  string
	KEYBOARD_SWITCH_KEY_NAME_S map[string]bool //键盘切换映射的按键集合
	view_range_limited         bool            //视角是否有界
	map_switch_signal          chan bool
	measure_sensitivity_mode   bool  //计算模式
	total_move_x               int32 //视角总移动距离x
	total_move_y               int32 //视角总移动距离y
	wheel_shift_enable         bool  //启用shift轮盘
	wheel_shift_switch_enable  bool  //shift轮盘切换 or 长按
	wheel_shift_range          int32

	// --- [修改 V1.1.1] ---
	key_jitter_enable    bool  // [新] 按键抖动开关
	key_jitter_amount_px int32 // [新] 按键抖动幅度(像素)
	// --- [修改 V1.1.1] 结束 ---

	// --- [修改 V1.1.0] ---
	wheel_step_speed float64 // [修改 V1.0.0] 轮盘移动平滑速度 (原版 60)

	wheel_planet_enable       bool    // [新] 行星转圈开关
	wheel_planet_radius_px    int32   // [新] 行星转圈半径(像素)
	wheel_planet_speed        float64 // [新] 行星转圈速度 (弧度/帧)
	wheel_planet_random_start bool    // [新 V1.0.1] 行星随机出发点
	planet_angle              float64 // [新] 行星当前角度

	planet_jitter_enable     bool    // [新] 行星抖动开关
	planet_jitter_amount_px  int32   // [新] 行星抖动幅度(像素)
	planet_jitter_frequency  float64 // [修改 V1.1.0] 行星抖动频率 (Hz)
	planet_jitter_counter    int32   // [修改 V1.1.0] 抖动计数器
	planet_jitter_last_x     int32   // [修改 V1.1.0]
	planet_jitter_last_y     int32   // [修改 V1.1.0]
	planet_jitter_wait_frames int32   // [修改 V1.1.0]

	star_jitter_enable     bool    // [修改 V1.0.1] 恒星抖动开关 (原 wheel_jitter)
	star_jitter_amount_px  int32   // [修改 V1.0.1] 恒星抖动幅度(像素)
	star_jitter_frequency  float64 // [修改 V1.1.0] 恒星抖动频率 (Hz)
	star_jitter_counter    int32   // [修改 V1.1.0]
	star_jitter_last_x     int32   // [修改 V1.1.0]
	star_jitter_last_y     int32   // [修改 V1.1.0]
	star_jitter_wait_frames int32   // [修改 V1.1.0]

	path_jitter_enable     bool    // [新 V1.0.1] 路径抖动开关
	path_jitter_amount_px  int32   // [新 V1.0.1] 路径抖动幅度(像素)
	path_jitter_frequency  float64 // [新 V1.1.0] 路径抖动频率 (Hz)
	path_jitter_counter    int32   // [新 V1.1.0]
	path_jitter_last_x     int32   // [新 V1.1.0]
	path_jitter_last_y     int32   // [新 V1.1.0]
	path_jitter_wait_frames int32   // [新 V1.1.0]

	wheel_star_x int32 // [新 V1.0.0] 轮盘恒星X
	wheel_star_y int32 // [新 V1.0.0] 轮盘恒星Y
	// --- [修改 V1.1.0] 结束 ---
}

const (
	TouchActionRequire int8 = 0
	TouchActionRelease int8 = 1
	TouchActionMove    int8 = 2
)

const (
	TouchActionResetResolution int8 = 3
)

const (
	UInput_mouse_move  int8 = 0
	UInput_mouse_btn   int8 = 1
	UInput_mouse_wheel int8 = 2
	UInput_key_event   int8 = 3
)

const (
	DOWN int32 = 1
	UP   int32 = 0
)

var UDF map[int32](string) = map[int32](string){
	DOWN: "🟢",
	UP:   "🔴",
}

const (
	Wheel_action_move    int8 = 1
	Wheel_action_release int8 = 0
)

const (
	touch_pos_scale uint8 = 8
	// touch_pos_scale uint8 = 0
)

// [修改 V1.1.0] 250Hz 循环
const (
	MAIN_LOOP_HZ        = 250.0
	MAIN_LOOP_NS_DOUBLE = float64(time.Second) / MAIN_LOOP_HZ
	MAIN_LOOP_NS_INT    = int64(MAIN_LOOP_NS_DOUBLE)
)

var HAT_D_U map[string]([]int32) = map[string]([]int32){
	"0.5_1.0": []int32{1, DOWN},
	"0.5_0.0": []int32{0, DOWN},
	"1.0_0.5": []int32{1, UP},
	"0.0_0.5": []int32{0, UP},
}

var HAT0_KEY_NAME map[string][]string = map[string][]string{
	"HAT0X": {"BTN_DPAD_LEFT", "BTN_DPAD_RIGHT"},
	"HAT0Y": {"BTN_DPAD_UP", "BTN_DPAD_DOWN"},
}

// [修改 V1.1.0] 计算频率应等待的帧数
func freq_to_wait_frames(freq float64) int32 {
	if freq <= 0 {
		return 99999999 // 几乎等于禁用
	}
	wait_frames := int32(MAIN_LOOP_HZ / freq)
	if wait_frames < 1 {
		return 1 // 最快每帧一次
	}
	return wait_frames
}

// [修改 V1.1.0] 基础抖动函数 (保持不变)
func get_jitter_offset(amount int32) int32 {
	if amount <= 0 {
		return 0
	}
	// rand.Int31n(n) returns a non-negative pseudo-random int32 in [0,n).
	// To get a range [-amount, +amount], we need n = 2*amount + 1
	// The result range is [0, 2*amount]. Subtract amount to get [-amount, amount].
	return (rand.Int31n(2*amount + 1)) - amount
}


// [修改 V1.1.0] 带频率的抖动函数
func (self *TouchHandler) get_jitter_offset_by_frequency(
	amount int32,
	wait_frames int32,
	counter *int32,
	last_x *int32,
	last_y *int32,
) (int32, int32) {
	if amount <= 0 {
		return 0, 0
	}

	*counter++ // 计数器增加

	// 如果计数器未达到等待帧数，返回上一次的抖动值
	if *counter < wait_frames {
		return *last_x, *last_y
	}

	// 达到帧数，重置计数器并计算新的抖动值
	*counter = 0
	*last_x = get_jitter_offset(amount)
	*last_y = get_jitter_offset(amount)
	return *last_x, *last_y
}

// --- [新 V1.1.1] 按键抖动应用函数 ---
func (self *TouchHandler) apply_key_jitter(x int32, y int32) (int32, int32) {
	if !self.key_jitter_enable || self.key_jitter_amount_px == 0 {
		return x, y
	}
	// 按键抖动不需要频率，每次都计算
	return x + get_jitter_offset(self.key_jitter_amount_px), y + get_jitter_offset(self.key_jitter_amount_px)
}
// --- [新 V1.1.1] 结束 ---

func InitTouchHandler(
	mapperFilePath string,
	events chan *event_pack,
	touch_control_func touch_control_func,
	u_input chan *u_input_control_pack,
	view_range_limited bool,
	map_switch_signal chan bool,
	measure_sensitivity_mode bool,
) *TouchHandler {
	rand.Seed(time.Now().UnixNano())

	//检查mapperFilePath文件是否存在
	if _, err := os.Stat(mapperFilePath); os.IsNotExist(err) {
		logger.Errorf("没有找到映射配置文件 : %s ", mapperFilePath)
		os.Exit(1)
	} else {
		logger.Infof("使用映射配置文件 : %s ", mapperFilePath)
	}

	content, _ := ioutil.ReadFile(mapperFilePath)
	config_json, _ := simplejson.NewJson(content)

	joystickInfo := make(map[string]*simplejson.Json)
	//插入远程遥控的手柄信息
	rjsJson := []byte(`{
    "DEADZONE": {
        "LS": [
            0.05,
            0.05
        ],
        "RS": [
            0.05,
            0.05
        ]
    },
    "ABS": {
        "7": {
            "name": "HAT0Y",
            "range": [
                -1,
                1
            ],
            "reverse": false
        },
        "6": {
            "name": "HAT0X",
            "range": [
                -1,
                1
            ],
            "reverse": false
        },
        "0": {
            "name": "LS_X",
            "range": [
                -32767,
                32767
            ],
            "reverse": false
        },
        "1": {
            "name": "LS_Y",
            "range": [
                -32767,
                32767
            ],
            "reverse": false
        },
		"2": {
            "name": "RS_X",
            "range": [
                -32767,
                32767
            ],
            "reverse": false
        },
        "3": {
            "name": "RS_Y",
            "range": [
                -32767,
				32767
            ],
            "reverse": false
        },
        "4": {
            "name": "LT",
            "range": [
                -1023,
                1023
            ],
            "reverse": false
        },
        "5": {
            "name": "RT",
            "range": [
                -1023,
                1023
            ],
            "reverse": false
        }
        
    },
    "BTN": {
        "0": "BTN_A",
        "1": "BTN_B",
        "2": "BTN_X",
        "3": "BTN_Y",
        "8": "BTN_LS",
        "9": "BTN_RS",
        "4": "BTN_LB",
        "5": "BTN_RB",
        "6": "BTN_SELECT",
        "7": "BTN_START",
        "10": "BTN_HOME"
    },
    "MAP_KEYBOARD": {
        "BTN_LT": "BTN_RIGHT",
        "BTN_RT": "BTN_LEFT",
        "BTN_DPAD_UP": "KEY_UP",
        "BTN_DPAD_LEFT": "KEY_LEFT",
        "BTN_DPAD_RIGHT": "KEY_RIGHT",
        "BTN_DPAD_DOWN": "KEY_DOWN",
        "BTN_A": "KEY_ENTER",
        "BTN_B": "KEY_BACK",
        "BTN_SELECT": "KEY_COMPOSE",
        "BTN_THUMBL": "KEY_HOME"
    }
}`)
	rjsJsonObj, err := simplejson.NewJson(rjsJson)
	if err != nil {
		logger.Errorf("Failed to parse rjs joystick config: %v", err)
		os.Exit(1)
	}
	joystickInfo["rjs"] = rjsJsonObj
	//check if dir ./joystickInfos exists
	path, _ := exec.LookPath(os.Args[0])
	abs, _ := filepath.Abs(path)
	workingDir, _ := filepath.Split(abs)
	joystickInfosDir := filepath.Join(workingDir, "joystickInfos")
	if _, err := os.Stat(joystickInfosDir); os.IsNotExist(err) {
		logger.Warnf("%s 文件夹不存在,没有载入任何手柄配置文件", joystickInfosDir)
	} else {
		files, _ := ioutil.ReadDir(joystickInfosDir)
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if file.Name()[len(file.Name())-5:] != ".json" {
				continue
			}
			// content, _ := ioutil.ReadFile("./joystickInfos/" + file.Name())
			content, _ := ioutil.ReadFile(filepath.Join(joystickInfosDir, file.Name()))
			info, _ := simplejson.NewJson(content)
			joystickInfo[file.Name()[:len(file.Name())-5]] = info
			logger.Infof("手柄配置文件已载入 : %s", file.Name())
		}
	}

	// logger.Infof("joystickInfo:%v", joystickInfo)

	abs_last_map := sync.Map{}

	abs_last_map.Store("HAT0X", 0.5)
	abs_last_map.Store("HAT0Y", 0.5)
	abs_last_map.Store("LT", 0.0)
	abs_last_map.Store("RT", 0.0)
	abs_last_map.Store("LS_X", 0.5)
	abs_last_map.Store("LS_Y", 0.5)
	abs_last_map.Store("RS_X", 0.5)
	abs_last_map.Store("RS_Y", 0.5)

	screenSizeX := config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt(3200) // [修改 V1.0.0] 添加默认值
	screenSizeY := config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt(1440) // [修改 V1.0.0] 添加默认值

	KEYBOARD_SWITCH_KEY_NAME_S := make(map[string]bool)
	for _, key := range config_json.Get("MOUSE").Get("SWITCH_KEYS").MustStringArray() {
		if key != "" {
			KEYBOARD_SWITCH_KEY_NAME_S[key] = true
		} else {
			logger.Warnf("映射配置文件中有空的键盘切换按键,请检查配置文件")
		}
	}

	// --- [修改 V1.1.2] 安全加载 KEY_JITTER 配置 ---
	var key_jitter_enable_val bool
	var key_jitter_amount_px_val int32
	if keyJitterJSON, ok := config_json.CheckGet("KEY_JITTER"); ok {
		key_jitter_enable_val = keyJitterJSON.Get("ENABLE").MustBool(true)
		key_jitter_amount_val := keyJitterJSON.Get("AMOUNT").MustFloat64(0.003)
		key_jitter_amount_px_val = int32(key_jitter_amount_val * float64(screenSizeX))
	} else {
		// 如果 KEY_JITTER 不存在，使用默认值
		key_jitter_enable_val = true
		key_jitter_amount_px_val = int32(0.003 * float64(screenSizeX))
	}
	// --- [修改 V1.1.2] 结束 ---


	// --- [修改 V1.1.0] 加载轮盘高级配置 ---
	wheel_step_speed_val := config_json.Get("WHEEL").Get("STEP_SPEED").MustFloat64(60) // [修改 V1.0.0]

	// 行星
	wheel_planet_enable_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("ENABLE").MustBool(false)
	wheel_planet_radius_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RADIUS").MustFloat64(0.015)
	wheel_planet_radius_px_val := int32(wheel_planet_radius_val * float64(screenSizeX)) // 用屏幕宽度计算像素
	wheel_planet_speed_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("SPEED").MustFloat64(1.5)
	wheel_planet_random_start_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RANDOM_START_ENABLE").MustBool(false) // [新 V1.0.1]

	// 行星抖动
	planet_jitter_enable_val := config_json.Get("WHEEL").Get("PLANET_JITTER").Get("ENABLE").MustBool(false)
	planet_jitter_amount_val := config_json.Get("WHEEL").Get("PLANET_JITTER").Get("AMOUNT").MustFloat64(0.005)
	planet_jitter_amount_px_val := int32(planet_jitter_amount_val * float64(screenSizeX))
	planet_jitter_freq_val := config_json.Get("WHEEL").Get("PLANET_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [修改 V1.1.0]
	planet_jitter_wait_frames_val := freq_to_wait_frames(planet_jitter_freq_val)                             // [修改 V1.1.0]

	// 恒星抖动 (V1.0.1 重命名)
	star_jitter_enable_val := config_json.Get("WHEEL").Get("STAR_JITTER").Get("ENABLE").MustBool(false)
	star_jitter_amount_val := config_json.Get("WHEEL").Get("STAR_JITTER").Get("AMOUNT").MustFloat64(0.002)
	star_jitter_amount_px_val := int32(star_jitter_amount_val * float64(screenSizeX))
	star_jitter_freq_val := config_json.Get("WHEEL").Get("STAR_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [修改 V1.1.0]
	star_jitter_wait_frames_val := freq_to_wait_frames(star_jitter_freq_val)                             // [修改 V1.1.0]

	// 路径抖动 (V1.0.1 新增)
	path_jitter_enable_val := config_json.Get("WHEEL").Get("PATH_JITTER").Get("ENABLE").MustBool(false)
	path_jitter_amount_val := config_json.Get("WHEEL").Get("PATH_JITTER").Get("AMOUNT").MustFloat64(0.002)
	path_jitter_amount_px_val := int32(path_jitter_amount_val * float64(screenSizeX))
	path_jitter_freq_val := config_json.Get("WHEEL").Get("PATH_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [新 V1.1.0]
	path_jitter_wait_frames_val := freq_to_wait_frames(path_jitter_freq_val)                           // [新 V1.1.0]

	// --- [修改 V1.1.0] 加载结束 ---

	return &TouchHandler{
		events:             events,
		touch_control_func: touch_control_func,
		u_input:            u_input,
		map_on:             false, //false
		view_id:            -1,
		wheel_id:           -1,
		allocated_id:       make([]bool, 12),
		// ^^^ 是可以创建超过12个的 只是不显示白点罢了
		config:         config_json,
		joystickInfo:   joystickInfo,
		screen_x:       int32(screenSizeX << touch_pos_scale),
		screen_y:       int32(screenSizeY << touch_pos_scale),
		rel_screen_x:   int32(screenSizeX),
		rel_screen_y:   int32(screenSizeY),
		view_init_x:    int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)),
		view_init_y:    int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)),
		view_current_x: int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)),
		view_current_y: int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)),
		view_speed_x:   int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(0).MustFloat64()),
		view_speed_y:   int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(1).MustFloat64()),
		// rs_speed_x:     config_json.Get("MOUSE").Get("RS_SPEED").GetIndex(0).MustFloat64(),
		// rs_speed_y:     config_json.Get("MOUSE").Get("RS_SPEED").GetIndex(1).MustFloat64(),
		rs_speed_x:   32,
		rs_speed_y:   32,
		wheel_init_x: int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX)),
		wheel_init_y: int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY)),
		wheel_range:  int32(config_json.Get("WHEEL").Get("RANGE").MustFloat64() * float64(screenSizeX)),
		wheel_wasd: []string{
			config_json.Get("WHEEL").Get("WASD").GetIndex(0).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(1).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(2).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(3).MustString(),
		},
		view_lock:               sync.Mutex{},
		wheel_lock:              sync.Mutex{},
		touch_control_lock:      sync.Mutex{},
		auto_release_view_count: 0,
		abs_last:                abs_last_map,
		using_joystick_name:     "",
		ls_wheel_released:       true,
		wasd_wheel_released:     true,
		wasd_wheel_last_x:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX)),
		wasd_wheel_last_y:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY)),
		wasd_up_down_statues:    make([]bool, 5), //放置wasd的状态与shift启用下，shift的状态
		key_action_state_save:   sync.Map{},
		BTN_SELECT_UP_DOWN:      0,
		// KEYBOARD_SWITCH_KEY_NAME:  config_json.Get("MOUSE").Get("SWITCH_KEY").MustString(),
		KEYBOARD_SWITCH_KEY_NAME_S: KEYBOARD_SWITCH_KEY_NAME_S,
		view_range_limited:         view_range_limited,
		map_switch_signal:          map_switch_signal,
		measure_sensitivity_mode:   measure_sensitivity_mode,
		wheel_shift_enable:         config_json.Get("WHEEL").Get("SHIFT_RANGE_ENABLE").MustBool(false), // [修改 V1.0.0] 添加默认值
		wheel_shift_switch_enable:  config_json.Get("WHEEL").Get("SHIFT_RANGE_SWITCH_ENABLE").MustBool(false), // [修改 V1.0.0] 添加默认值
		wheel_shift_range:          int32(config_json.Get("WHEEL").Get("SHIFT_RANGE").MustFloat64() * float64(screenSizeX)),

		// --- [修改 V1.1.2] 初始化 KEY_JITTER ---
		key_jitter_enable:    key_jitter_enable_val,
		key_jitter_amount_px: key_jitter_amount_px_val,
		// --- [修改 V1.1.2] 结束 ---

		// --- [修改 V1.1.0] 初始化轮盘高级配置 ---
		wheel_step_speed: wheel_step_speed_val, // [修改 V1.0.0]

		wheel_planet_enable:       wheel_planet_enable_val,
		wheel_planet_radius_px:    wheel_planet_radius_px_val,
		wheel_planet_speed:        wheel_planet_speed_val / MAIN_LOOP_HZ, // [修改 V1.0.0] 转换为 弧度/帧
		wheel_planet_random_start: wheel_planet_random_start_val,       // [新 V1.0.1]
		planet_angle:              0,

		planet_jitter_enable:     planet_jitter_enable_val,
		planet_jitter_amount_px:  planet_jitter_amount_px_val,
		planet_jitter_frequency:  planet_jitter_freq_val,      // [修改 V1.1.0]
		planet_jitter_wait_frames: planet_jitter_wait_frames_val, // [修改 V1.1.0]
		planet_jitter_counter:    0,
		planet_jitter_last_x:     0,
		planet_jitter_last_y:     0,

		star_jitter_enable:     star_jitter_enable_val,
		star_jitter_amount_px:  star_jitter_amount_px_val,
		star_jitter_frequency:  star_jitter_freq_val,      // [修改 V1.1.0]
		star_jitter_wait_frames: star_jitter_wait_frames_val, // [修改 V1.1.0]
		star_jitter_counter:    0,
		star_jitter_last_x:     0,
		star_jitter_last_y:     0,

		path_jitter_enable:     path_jitter_enable_val,
		path_jitter_amount_px:  path_jitter_amount_px_val,
		path_jitter_frequency:  path_jitter_freq_val,      // [新 V1.1.0]
		path_jitter_wait_frames: path_jitter_wait_frames_val, // [新 V1.1.0]
		path_jitter_counter:    0,
		path_jitter_last_x:     0,
		path_jitter_last_y:     0,

		wheel_star_x: 0, // [新 V1.0.0]
		wheel_star_y: 0, // [新 V1.0.0]
		// --- [修改 V1.1.0] 初始化结束 ---
	}
}

func (self *TouchHandler) reloadConfigure(mapperFilePath string) {
	if self.map_on {
		self.switch_map_mode()
	}
	if _, err := os.Stat(mapperFilePath); os.IsNotExist(err) {
		logger.Errorf("没有找到映射配置文件 : %s ", mapperFilePath)
		os.Exit(1)
	} else {
		logger.Infof("使用映射配置文件 : %s ", mapperFilePath)
	}
	content, _ := ioutil.ReadFile(mapperFilePath)
	config_json, _ := simplejson.NewJson(content)
	screenSizeX := config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt(3200) // [修改 V1.0.0] 添加默认值
	screenSizeY := config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt(1440) // [修改 V1.0.0] 添加默认值
	self.config = config_json
	self.screen_x = int32(screenSizeX << touch_pos_scale)
	self.screen_y = int32(screenSizeY << touch_pos_scale)
	self.rel_screen_x = int32(screenSizeX)
	self.rel_screen_y = int32(screenSizeY)
	self.view_init_x = int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale))
	self.view_init_y = int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale))
	self.view_current_x = int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale))
	self.view_current_y = int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale))
	self.view_speed_x = int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(0).MustFloat64())
	self.view_speed_y = int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(1).MustFloat64())
	self.wheel_init_x = int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX))
	self.wheel_init_y = int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY))
	self.wheel_range = int32(config_json.Get("WHEEL").Get("RANGE").MustFloat64() * float64(screenSizeX))
	self.wheel_wasd = []string{
		config_json.Get("WHEEL").Get("WASD").GetIndex(0).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(1).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(2).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(3).MustString(),
	}
	self.wasd_wheel_last_x = int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX))
	self.wasd_wheel_last_y = int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY))
	// self.KEYBOARD_SWITCH_KEY_NAME = config_json.Get("MOUSE").Get("SWITCH_KEY").MustString()
	// self.KEYBOARD_SWITCH_KEY_NAME_S = config_json.Get("MOUSE").Get("SWITCH_KEYS").MustStringArray()
	self.KEYBOARD_SWITCH_KEY_NAME_S = make(map[string]bool)
	for _, key := range config_json.Get("MOUSE").Get("SWITCH_KEYS").MustStringArray() {
		if key != "" {
			self.KEYBOARD_SWITCH_KEY_NAME_S[key] = true
		} else {
			logger.Warnf("映射配置文件中有空的键盘切换按键,请检查配置文件")
		}
	}

	self.wheel_shift_enable = config_json.Get("WHEEL").Get("SHIFT_RANGE_ENABLE").MustBool(false) // [修改 V1.0.0] 添加默认值
	self.wheel_shift_switch_enable = config_json.Get("WHEEL").Get("SHIFT_RANGE_SWITCH_ENABLE").MustBool(false) // [修改 V1.0.0] 添加默认值
	self.wheel_shift_range = int32(config_json.Get("WHEEL").Get("SHIFT_RANGE").MustFloat64() * float64(screenSizeX))

	// --- [修改 V1.1.2] 安全加载 KEY_JITTER ---
	if keyJitterJSON, ok := config_json.CheckGet("KEY_JITTER"); ok {
		self.key_jitter_enable = keyJitterJSON.Get("ENABLE").MustBool(true)
		self.key_jitter_amount_px = int32(keyJitterJSON.Get("AMOUNT").MustFloat64(0.003) * float64(screenSizeX))
	} else {
		self.key_jitter_enable = true
		self.key_jitter_amount_px = int32(0.003 * float64(screenSizeX))
	}
	// --- [修改 V1.1.2] 结束 ---

	// --- [修改 V1.1.0] 重新加载轮盘高级配置 ---
	self.wheel_step_speed = config_json.Get("WHEEL").Get("STEP_SPEED").MustFloat64(60) // [修改 V1.0.0]

	// 行星
	self.wheel_planet_enable = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("ENABLE").MustBool(false)
	self.wheel_planet_radius_px = int32(config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RADIUS").MustFloat64(0.015) * float64(screenSizeX))
	self.wheel_planet_speed = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("SPEED").MustFloat64(1.5) / MAIN_LOOP_HZ // [修改 V1.0.0] 转换为 弧度/帧
	self.wheel_planet_random_start = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RANDOM_START_ENABLE").MustBool(false) // [新 V1.0.1]

	// 行星抖动
	self.planet_jitter_enable = config_json.Get("WHEEL").Get("PLANET_JITTER").Get("ENABLE").MustBool(false)
	self.planet_jitter_amount_px = int32(config_json.Get("WHEEL").Get("PLANET_JITTER").Get("AMOUNT").MustFloat64(0.005) * float64(screenSizeX))
	planet_jitter_freq_val := config_json.Get("WHEEL").Get("PLANET_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [修改 V1.1.0]
	self.planet_jitter_frequency = planet_jitter_freq_val                                                   // [修改 V1.1.0]
	self.planet_jitter_wait_frames = freq_to_wait_frames(planet_jitter_freq_val)                            // [修改 V1.1.0]

	// 恒星抖动 (V1.0.1 重命名)
	self.star_jitter_enable = config_json.Get("WHEEL").Get("STAR_JITTER").Get("ENABLE").MustBool(false)
	self.star_jitter_amount_px = int32(config_json.Get("WHEEL").Get("STAR_JITTER").Get("AMOUNT").MustFloat64(0.002) * float64(screenSizeX))
	star_jitter_freq_val := config_json.Get("WHEEL").Get("STAR_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [修改 V1.1.0]
	self.star_jitter_frequency = star_jitter_freq_val                                                  // [修改 V1.1.0]
	self.star_jitter_wait_frames = freq_to_wait_frames(star_jitter_freq_val)                           // [修改 V1.1.0]

	// 路径抖动 (V1.0.1 新增)
	self.path_jitter_enable = config_json.Get("WHEEL").Get("PATH_JITTER").Get("ENABLE").MustBool(false)
	self.path_jitter_amount_px = int32(config_json.Get("WHEEL").Get("PATH_JITTER").Get("AMOUNT").MustFloat64(0.002) * float64(screenSizeX))
	path_jitter_freq_val := config_json.Get("WHEEL").Get("PATH_JITTER").Get("FREQUENCY").MustFloat64(1.0) // [新 V1.1.0]
	self.path_jitter_frequency = path_jitter_freq_val                                                  // [新 V1.1.0]
	self.path_jitter_wait_frames = freq_to_wait_frames(path_jitter_freq_val)                           // [新 V1.1.0]

	// --- [修改 V1.1.0] 重新加载结束 ---
}

func (self *TouchHandler) touch_require(x int32, y int32, scale uint8) int32 {

	for i, v := range self.allocated_id {
		if !v {
			self.allocated_id[i] = true
			self.send_touch_control_pack(TouchActionRequire, int32(i), x<<scale, y<<scale)
			logger.Debugf("touch require (%v,%v) <= [%v]", x, y, i)
			return int32(i)
		}
	}
	return -1
}

func (self *TouchHandler) touch_release(id int32) int32 {
	logger.Debugf("touch release [%v]", id)
	if id != -1 {
		self.allocated_id[int(id)] = false
		self.send_touch_control_pack(TouchActionRelease, id, -1, -1)
	}
	return -1
}

func (self *TouchHandler) touch_move(id int32, x int32, y int32, scale uint8) {
	logger.Debugf("touch move to (%v,%v) [%v]", x, y, id)
	if id != -1 {
		self.send_touch_control_pack(TouchActionMove, id, x<<scale, y<<scale)
	}
}

func (self *TouchHandler) u_input_control(action int8, arg1 int32, arg2 int32) {
	self.u_input <- &u_input_control_pack{
		action: action,
		arg1:   arg1,
		arg2:   arg2,
	}
}

func (self *TouchHandler) send_touch_control_pack(action int8, id int32, x int32, y int32) {
	self.touch_control_lock.Lock()
	defer self.touch_control_lock.Unlock()
	self.touch_control_func(touch_control_pack{
		action:   action,
		id:       id,
		x:        x,
		y:        y,
		screen_x: self.screen_x,
		screen_y: self.screen_y,
	})
}

// [修改 V1.0.3] 提取出行星位置计算，用于修复BUG
func (self *TouchHandler) get_planet_pos() (int32, int32) {
	// 1. 获取恒星位置 (恒星抖动)
	star_jit_x, star_jit_y := self.get_jitter_offset_by_frequency(
		self.star_jitter_amount_px,
		self.star_jitter_wait_frames,
		&self.star_jitter_counter,
		&self.star_jitter_last_x,
		&self.star_jitter_last_y,
	)
	final_star_x := self.wheel_star_x + star_jit_x
	final_star_y := self.wheel_star_y + star_jit_y

	// 2. 如果行星功能关闭，直接返回恒星位置
	if !self.wheel_planet_enable {
		return final_star_x, final_star_y
	}

	// 3. 计算行星位置
	self.planet_angle += self.wheel_planet_speed // 增加角度
	if self.planet_angle > 2*math.Pi {
		self.planet_angle -= 2 * math.Pi
	}

	planet_x := final_star_x + int32(float64(self.wheel_planet_radius_px)*math.Cos(self.planet_angle))
	planet_y := final_star_y + int32(float64(self.wheel_planet_radius_px)*math.Sin(self.planet_angle))

	// 4. 计算行星抖动
	planet_jit_x, planet_jit_y := self.get_jitter_offset_by_frequency(
		self.planet_jitter_amount_px,
		self.planet_jitter_wait_frames,
		&self.planet_jitter_counter,
		&self.planet_jitter_last_x,
		&self.planet_jitter_last_y,
	)

	// 5. 返回最终位置
	return planet_x + planet_jit_x, planet_y + planet_jit_y
}

// [新 V1.0.0] 行星循环
func (self *TouchHandler) loop_handel_wheel_planet() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			// 仅在轮盘被按下时 (wheel_id != -1) 才执行移动
			if self.wheel_id != -1 {
				self.wheel_lock.Lock()
				// [修改 V1.0.3] 检查 self.wheel_id 确保触点未被释放
				if self.wheel_id != -1 {
					// [修改 V1.0.3] 使用 get_planet_pos() 获取最终位置
					final_x, final_y := self.get_planet_pos()
					self.touch_move(self.wheel_id, final_x, final_y, touch_pos_scale)
				}
				self.wheel_lock.Unlock()
			}
			// 必须休眠，否则会占用100% CPU
			time.Sleep(time.Duration(MAIN_LOOP_NS_INT) * time.Nanosecond) // 250Hz
		}
	}
}

func (self *TouchHandler) loop_handel_rs_move() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			rs_x, rs_y := self.getStick("RS")
			if rs_x != 0.5 || rs_y != 0.5 {
				if self.map_on {
					self.handel_view_move(int32((rs_x-0.5)*self.rs_speed_x), int32((rs_y-0.5)*self.rs_speed_y))
				} else {
					self.u_input_control(UInput_mouse_move, int32((rs_x-0.5)*24), int32((rs_y-0.5)*24))
				}
			}
			time.Sleep(time.Duration(MAIN_LOOP_NS_INT) * time.Nanosecond) // [修改 V1.1.0] 250HZ
		}
	}
}

func (self *TouchHandler) handel_view_move(offset_x int32, offset_y int32) { //视角移动
	self.view_lock.Lock()
	defer self.view_lock.Unlock()
	if self.measure_sensitivity_mode {
		self.total_move_x += offset_x
		self.total_move_y += offset_y
		logger.Infof("total_move_x:%v\ttotal_move_y:%v", self.total_move_x, self.total_move_y)
	}
	self.auto_release_view_count = 0
	if self.view_id == -1 {
		// [修改 V1.1.0] 移除 rand_offset()，视角不需要抖动
		self.view_current_x = self.view_init_x // + rand_offset()<<touch_pos_scale
		self.view_current_y = self.view_init_y // + rand_offset()<<touch_pos_scale
		self.view_id = self.touch_require(self.view_current_x, self.view_current_y, 0)
	}
	self.view_current_x += offset_x * self.view_speed_x
	self.view_current_y += offset_y * self.view_speed_y
	if self.view_range_limited { //有界 or 无界 即 使用eventX 还是 inputManager
		if self.view_current_x <= 0 || self.view_current_x >= self.screen_x || self.view_current_y <= 0 || self.view_current_y >= self.screen_y {
			//测试了两个软件
			//都可以同时两个触摸点控制视角
			//所以这里超出范围时候逻辑修改了
			//原本的点的目标超出了 但是暂时不释放
			//此时去申请一个新的触控点来执行本次滑动操作
			//然后再将原本的触控点释放 并将新的触控点设置为当前控制用的触控点
			//即从原本的瞬间松开再按下 改为了按下新的再松开
			
			// [修改 V1.1.0] 移除 rand_offset()
			self.view_current_x = self.view_init_x // + rand_offset()
			self.view_current_y = self.view_init_y // + rand_offset()
			tmp_view_id := self.touch_require(self.view_current_x, self.view_current_y, 0)
			self.view_current_x += offset_x * self.view_speed_x //用的时直接写event坐标系
			self.view_current_y += offset_y * self.view_speed_y
			self.touch_move(tmp_view_id, self.view_current_x, self.view_current_y, 0)
			self.touch_release(self.view_id)
			self.view_id = tmp_view_id
		} else {
			self.touch_move(self.view_id, self.view_current_x, self.view_current_y, 0)
		}
	} else {
		self.touch_move(self.view_id, self.view_current_x, self.view_current_y, 0)
	}
}

func (self *TouchHandler) auto_handel_view_release(timeout int) { //视角释放
	if timeout == 0 {
		return
	} else {
		for {
			select {
			case <-global_close_signal:
				return
			default:
				self.view_lock.Lock()
				if self.view_id != -1 {
					self.auto_release_view_count += 1
					if self.auto_release_view_count > int32(timeout/50) { //200ms不动 则释放
						self.auto_release_view_count = 0
						self.view_id = self.touch_release(self.view_id)
					}
				}
				self.view_lock.Unlock()
				time.Sleep(time.Duration(50) * time.Millisecond)
			}
		}
	}
}

// [修改 V1.0.3] 轮盘核心逻辑重构
func (self *TouchHandler) handel_wheel_action(action int8, abs_x int32, abs_y int32) {
	self.wheel_lock.Lock()
	defer self.wheel_lock.Unlock()

	if action == Wheel_action_release { // 释放
		if self.wheel_id != -1 {
			self.wheel_id = self.touch_release(self.wheel_id)
		}
		// 重置恒星位置到中心
		self.wheel_star_x = self.wheel_init_x
		self.wheel_star_y = self.wheel_init_y
	} else if action == Wheel_action_move { // 移动
		if self.wheel_id == -1 {
			// [新 V1.0.1] 随机出发点
			if self.wheel_planet_random_start {
				self.planet_angle = rand.Float64() * 2 * math.Pi
			} else {
				self.planet_angle = 0 // 重置角度
			}

			// [修复 V1.0.3]
			// 轮盘 *按下* 时，必须使用 *原始坐标* (abs_x, abs_y)
			// 而不是 get_planet_pos() 的计算结果
			// 否则触点会出现在奇怪的抖动或旋转后的位置
			self.wheel_id = self.touch_require(abs_x, abs_y, touch_pos_scale)

		}
		// 仅更新“恒星”位置
		// loop_handel_wheel_planet 循环会处理实际的 touch_move
		self.wheel_star_x = abs_x
		self.wheel_star_y = abs_y
	}
}

func (self *TouchHandler) get_wasd_now_target() (int32, int32) { //根据wasd当前状态 获取wasd滚轮的目标位置
	var x int32 = 0
	var y int32 = 0
	if self.wasd_up_down_statues[0] {
		y -= 1
	}
	if self.wasd_up_down_statues[2] {
		y += 1
	}
	if self.wasd_up_down_statues[1] {
		x -= 1
	}
	if self.wasd_up_down_statues[3] {
		x += 1
	}

	wheel_range := self.wheel_range
	if self.wasd_up_down_statues[4] {
		wheel_range = self.wheel_shift_range
	}

	if x*y == 0 {
		// logger.Warnf("%v  %v", self.wheel_init_x+x*wheel_range, self.wheel_init_y+y*wheel_range)
		return self.wheel_init_x + x*wheel_range, self.wheel_init_y + y*wheel_range
	} else {
		// logger.Warnf("%v  %v", self.wheel_init_x+x*wheel_range*707/1000, self.wheel_init_y+y*wheel_range*707/1000)
		return self.wheel_init_x + x*wheel_range*707/1000, self.wheel_init_y + y*wheel_range*707/1000
	}
}

// [修改 V1.0.0] 使用可配置的 wheel_step_speed
func (self *TouchHandler) update_wheel_xy(last_x, last_y, target_x, target_y int32) (int32, int32) {
	if last_x == target_x && last_y == target_y {
		return last_x, last_y
	} else {
		x_rest := target_x - last_x
		y_rest := target_y - last_y
		total_rest := int32(math.Sqrt(float64(x_rest*x_rest + y_rest*y_rest)))
		
		// 使用配置的 step_speed
		wheel_step_val_int32 := int32(self.wheel_step_speed)

		if total_rest <= wheel_step_val_int32 {
			return target_x, target_y
		} else {
			return last_x + x_rest*wheel_step_val_int32/total_rest, last_y + y_rest*wheel_step_val_int32/total_rest
		}
	}
}

func (self *TouchHandler) loop_handel_wasd_wheel() { //循环处理wasd映射轮盘并控制释放
	for {
		select {
		case <-global_close_signal:
			return
		default:
			wasd_wheel_target_x, wasd_wheel_target_y := self.get_wasd_now_target() //获取目标位置
			if self.wheel_init_x == wasd_wheel_target_x && self.wheel_init_y == wasd_wheel_target_y {
				self.wasd_wheel_released = true //如果wasd目标位置 等于 wasd轮盘初始位置 则认为轮盘释放
				// [修改 V1.1.0] 移除 rand_offset()
				self.wasd_wheel_last_x = self.wheel_init_x // + rand_offset()
				self.wasd_wheel_last_y = self.wheel_init_y // + rand_offset()
			} else {
				self.wasd_wheel_released = false
				if self.wasd_wheel_last_x != wasd_wheel_target_x || self.wasd_wheel_last_y != wasd_wheel_target_y {
					// [修改 V1.0.0] 使用 update_wheel_xy
					self.wasd_wheel_last_x, self.wasd_wheel_last_y = self.update_wheel_xy(self.wasd_wheel_last_x, self.wasd_wheel_last_y, wasd_wheel_target_x, wasd_wheel_target_y)

					// [新 V1.1.0] 计算路径抖动
					path_jit_x, path_jit_y := self.get_jitter_offset_by_frequency(
						self.path_jitter_amount_px,
						self.path_jitter_wait_frames,
						&self.path_jitter_counter,
						&self.path_jitter_last_x,
						&self.path_jitter_last_y,
					)

					// [修改 V1.0.1]
					self.handel_wheel_action(Wheel_action_move, self.wasd_wheel_last_x+path_jit_x, self.wasd_wheel_last_y+path_jit_y)
				}
			}
			if self.wheel_id != -1 && self.wasd_wheel_released && self.ls_wheel_released {
				self.handel_wheel_action(Wheel_action_release, -1, -1) //wheel当前按下 且两个标记都释放 则释放
			}
			time.Sleep(time.Duration(MAIN_LOOP_NS_INT) * time.Nanosecond) // [修改 V1.1.0] 250HZ
		}
	}
}

func (self *TouchHandler) quick_click(keyname string) {
	self.handel_key_up_down(keyname, DOWN, "MOUSE_WHEEL")
	time.Sleep(time.Duration(50) * time.Millisecond)
	self.handel_key_up_down(keyname, UP, "MOUSE_WHEEL")
}

func (self *TouchHandler) handel_rel_event(x int32, y int32, HWhell int32, Wheel int32) {
	if x != 0 || y != 0 {
		if self.map_on {
			self.handel_view_move(x, y)
		} else {
			self.u_input_control(UInput_mouse_move, x, y)
		}
	}

	if HWhell != 0 {
		if self.map_on {
			if HWhell > 0 {
				go self.quick_click("REL_HWHEEL_UP")
			} else if HWhell < 0 {
				go self.quick_click("REL_HWHEEL_DOWN")
			}
		} else {
			self.u_input_control(UInput_mouse_wheel, REL_HWHEEL, HWhell)
		}
	}
	if Wheel != 0 {
		if self.map_on {
			if Wheel > 0 {
				go self.quick_click("REL_WHEEL_UP") //纵向滚轮向上
			} else if Wheel < 0 {
				go self.quick_click("REL_WHEEL_DOWN") //纵向滚轮向下
			}
		} else {
			self.u_input_control(UInput_mouse_wheel, REL_WHEEL, Wheel)
		}
	}
}

func (self *TouchHandler) execute_key_action(start time.Time, key_name string, up_down int32, action *simplejson.Json, state interface{}) {
	action_type := action.Get("TYPE").MustString()
	if key_name == "REL_WHEEL_DOWN" || key_name == "REL_WHEEL_UP" || key_name == "REL_HWHEEL_DOWN" || key_name == "REL_HWHEEL_UP" {
		if action_type == "PRESS" || action_type == "AUTO_FIRE" || action_type == "MULT_PRESS" {
			logger.Errorf("鼠标滚轮无法使用动作类型:%v", action_type) //二次保证
		}
	}
	defer logger.Debugf("key[%s]%s\t%v\t%v", key_name, UDF[up_down], action, time.Since(start))
	switch action_type {
	case "PRESS": //按键的按下与释放直接映射为触屏的按下与释放
		if up_down == DOWN {
			x := int32(action.Get("POS").GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
			y := int32(action.Get("POS").GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
			// --- [修改 V1.1.1] 应用按键抖动 ---
			x_jit, y_jit := self.apply_key_jitter(x, y)
			self.key_action_state_save.Store(key_name, self.touch_require(x_jit, y_jit, touch_pos_scale))
			// --- [修改 V1.1.1] 结束 ---
		} else if up_down == UP {
			tid := state.(int32)
			self.touch_release(tid)
			self.key_action_state_save.Delete(key_name)
		}
	case "CLICK": //仅在按下的时候执行一次 不保存状态所以不响应down 也不会有down到这里
		if up_down == DOWN {
			go (func() {
				x := int32(action.Get("POS").GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				y := int32(action.Get("POS").GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				// --- [修改 V1.1.1] 应用按键抖动 ---
				x_jit, y_jit := self.apply_key_jitter(x, y)
				tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
				// --- [修改 V1.1.1] 结束 ---
				time.Sleep(time.Duration(8) * time.Millisecond) //8ms 120HZ下一次
				self.touch_release(tid)
			})()
		}

	case "AUTO_FIRE": //连发 按下开始 松开结束 按照设置的间隔 持续点击
		if up_down == DOWN {
			x := int32(action.Get("POS").GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
			y := int32(action.Get("POS").GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
			down_time := action.Get("INTERVAL").GetIndex(0).MustInt()
			interval_time := action.Get("INTERVAL").GetIndex(1).MustInt()
			self.key_action_state_save.Store(key_name, true)
			go (func() {
				for {
					// --- [修改 V1.1.1] 应用按键抖动 ---
					x_jit, y_jit := self.apply_key_jitter(x, y)
					tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
					// --- [修改 V1.1.1] 结束 ---
					time.Sleep(time.Duration(down_time) * time.Millisecond)
					self.touch_release(tid)
					time.Sleep(time.Duration(interval_time) * time.Millisecond)
					if running, ok := self.key_action_state_save.Load(key_name); !ok || running == false {
						break
					}
				}
				self.key_action_state_save.Delete(key_name)
			})()

		} else if up_down == UP {
			self.key_action_state_save.Store(key_name, false)
		}

	case "MULT_PRESS": //多点触摸 按照顺序按下 松开再反向松开 实现类似一键开镜开火
		if up_down == DOWN {
			tid_save := make([]int32, 0)
			release_signal := make(chan bool, 16)
			self.key_action_state_save.Store(key_name, release_signal)
			go (func() {
				for i := range action.Get("POS_S").MustArray() {
					x := int32(action.Get("POS_S").GetIndex(i).GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
					y := int32(action.Get("POS_S").GetIndex(i).GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
					// --- [修改 V1.1.1] 应用按键抖动 ---
					x_jit, y_jit := self.apply_key_jitter(x, y)
					tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
					// --- [修改 V1.1.1] 结束 ---
					tid_save = append(tid_save, tid)
					time.Sleep(time.Duration(8) * time.Millisecond) // 间隔8ms 是否需要延迟有待验证
				}
				<-release_signal
				self.key_action_state_save.Delete(key_name)
				for i := len(tid_save) - 1; i >= 0; i-- {
					self.touch_release(tid_save[i])
					time.Sleep(time.Duration(8) * time.Millisecond)
				}
			})()
		} else if up_down == UP {
			state.(chan bool) <- true
			//按下立即创建channel 并保存状态
			//松开拿到的channel 并发送信号
			//同时立即删除状态
			//即按下立即执行并等待释放,此过程中不响应按下但是可以响应多次松开 //缓冲区大小
			//而再松开后释放触摸过程中便可以再次响应按下
		}
	case "DRAG": //只响应一次按下  可同时多次触发
		if up_down == DOWN {
			go (func() {
				pos_len := len(action.Get("POS_S").MustArray())
				interval_time := action.Get("INTERVAL").GetIndex(0).MustInt()
				init_x := int32(action.Get("POS_S").GetIndex(0).GetIndex(0).MustFloat64() * float64(self.rel_screen_x))
				init_y := int32(action.Get("POS_S").GetIndex(0).GetIndex(1).MustFloat64() * float64(self.rel_screen_y))
				
				// --- [修改 V1.1.1] 应用按键抖动 ---
				init_x_jit, init_y_jit := self.apply_key_jitter(init_x, init_y)
				tid := self.touch_require(init_x_jit, init_y_jit, touch_pos_scale)
				// --- [修改 V1.1.1] 结束 ---
				
				time.Sleep(time.Duration(interval_time) * time.Millisecond)
				for index := 1; index < pos_len-1; index++ {
					x := int32(action.Get("POS_S").GetIndex(index).GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
					y := int32(action.Get("POS_S").GetIndex(index).GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
					// --- [修改 V1.1.1] 应用按键抖动 ---
					x_jit, y_jit := self.apply_key_jitter(x, y)
					self.touch_move(tid, x_jit, y_jit, touch_pos_scale)
					// --- [修改 V1.1.1] 结束 ---
					time.Sleep(time.Duration(interval_time) * time.Millisecond)
				}
				end_x := int32(action.Get("POS_S").GetIndex(pos_len-1).GetIndex(0).MustFloat64() * float64(self.rel_screen_x))
				end_y := int32(action.Get("POS_S").GetIndex(pos_len-1).GetIndex(1).MustFloat64() * float64(self.rel_screen_y))
				// --- [修改 V1.1.1] 应用按键抖动 ---
				end_x_jit, end_y_jit := self.apply_key_jitter(end_x, end_y)
				self.touch_move(tid, end_x_jit, end_y_jit, touch_pos_scale)
				// --- [修改 V1.1.1] 结束 ---
				self.touch_release(tid)
			})()
		} else if up_down == UP {

		}

	}
}

func (self *TouchHandler) switch_map_mode() {
	self.total_move_x = 0
	self.total_move_y = 0                           //总移动距离清零
	self.view_id = self.touch_release(self.view_id) //视角id释放

	self.key_action_state_save.Range(func(key, value interface{}) bool {
		self.execute_key_action(time.Now(), key.(string), UP, self.config.Get("KEY_MAPS").Get(key.(string)), value)
		logger.Infof("已释放key:%s", key.(string))
		return true
	})

	self.map_on = !self.map_on     //切换
	self.map_switch_signal <- true //发送信号到v_mouse切换显示

	// logger.Infof("map_on:%v", self.map_on)
	if self.map_on {
		logger.Info("映射[on]")
	} else {
		logger.Info("映射[off]")
	}
}

func (self *TouchHandler) handel_key_up_down(key_name string, up_down int32, dev_name string) {
	if key_name == "" {
		return
	}
	if key_name == "BTN_SELECT" {
		if up_down == DOWN || up_down == UP {
			self.BTN_SELECT_UP_DOWN = up_down
		}
	}
	if self.BTN_SELECT_UP_DOWN == DOWN {
		if key_name == "BTN_RS" && up_down == UP {
			self.switch_map_mode()
			return
		}
	}

	if self.KEYBOARD_SWITCH_KEY_NAME_S[key_name] {
		if up_down == UP {
			self.switch_map_mode()
		}
		return
	}

	if self.map_on {
		for i := 0; i < 4; i++ {
			if self.wheel_wasd[i] == key_name {
				if up_down == DOWN {
					self.wasd_up_down_statues[i] = true
				} else if up_down == UP {
					self.wasd_up_down_statues[i] = false
				}
				return
			}
		}
		if self.wheel_shift_enable && key_name == "KEY_LEFTSHIFT" {
			if self.wheel_shift_switch_enable { //切换模式
				if up_down == DOWN {
					self.wasd_up_down_statues[4] = !self.wasd_up_down_statues[4]
				}
			} else { //长按模式
				if up_down == DOWN {
					self.wasd_up_down_statues[4] = true
				} else if up_down == UP {
					self.wasd_up_down_statues[4] = false
				}
			}
			return
		}

		if self.measure_sensitivity_mode && up_down == UP {
			if key_name == "KEY_LEFT" {
				self.handel_view_move(-1, 0)
				return
			} else if key_name == "KEY_RIGHT" {
				self.handel_view_move(1, 0)
				return
			} else if key_name == "KEY_UP" {
				self.handel_view_move(0, -1)
				return
			} else if key_name == "KEY_DOWN" {
				self.handel_view_move(0, 1)
				return
			}
		}
		if action, ok := self.config.Get("KEY_MAPS").CheckGet(key_name); ok {
			state, ok := self.key_action_state_save.Load(key_name)
			if (up_down == UP && !ok) || (up_down == DOWN && ok) {
			} else {
				// logger.Debugf("key[%s]%s\t%v\t%v", key_name, UDF[up_down], action, state)
				self.execute_key_action(time.Now(), key_name, up_down, action, state)
			}
		} else {
			logger.Debugf("key[%s]\t无触屏映射", key_name)
		}
	} else {
		if jsconfig, ok := self.joystickInfo[dev_name]; ok {
			//如果是手柄 则检查是否设置了键盘映射
			if joystick_btn_map_key_name, ok := jsconfig.Get("MAP_KEYBOARD").CheckGet(key_name); ok {
				//有则映射到普通按键
				self.handel_key_up_down(joystick_btn_map_key_name.MustString(), up_down, dev_name+"_joystick_mapped")
			} else {
				logger.Debugf("joyStick[%s]\tkey[%s]\t无键盘映射", dev_name, key_name)
			}
		} else {
			if code, ok := friendly_name_2_keycode[key_name]; ok {
				//是合法按键 则输出
				self.u_input_control(UInput_key_event, int32(code), int32(up_down))
			}
		}
	}

}

func (self *TouchHandler) handel_key_events(events []*evdev.Event, dev_type dev_type, dev_name string) {
	if jsconfig, ok := self.joystickInfo[dev_name]; ok && dev_type == type_joystick {
		for _, event := range events {
			if key_name, ok := jsconfig.Get("BTN").CheckGet(strconv.Itoa(int(event.Code))); ok {
				self.handel_key_up_down(key_name.MustString(), event.Value, dev_name)
			} else {
				logger.Debugf("joyStick[%s]\t%d\t未知键码", dev_name, event.Code)
			}
		}
	} else {
		for _, event := range events {
			self.handel_key_up_down(GetKeyName(event.Code), event.Value, dev_name)
		}
	}
}

func (self *TouchHandler) getStick(stick_name string) (float64, float64) {
	if jsconfig, ok := self.joystickInfo[self.using_joystick_name]; ok {
		_x, _ := self.abs_last.Load(stick_name + "_X")
		_y, _ := self.abs_last.Load(stick_name + "_Y")
		x, y := _x.(float64), _y.(float64)
		deadZone_left := jsconfig.Get("DEADZONE").Get(stick_name).GetIndex(0).MustFloat64()
		deadZone_right := jsconfig.Get("DEADZONE").Get(stick_name).GetIndex(1).MustFloat64()
		if deadZone_left < x && x < deadZone_right && deadZone_left < y && y < deadZone_right {
			return 0.5, 0.5
		} else {
			return x, y
		}
	} else {
		return 0.5, 0.5
	}
}

func (self *TouchHandler) handel_abs_events(events []*evdev.Event, dev_type dev_type, dev_name string) {
	for _, event := range events {
		if jsconfig, ok := self.joystickInfo[dev_name]; ok && dev_type == type_joystick {
			abs_info := jsconfig.Get("ABS").Get(strconv.Itoa(int(event.Code)))
			name := abs_info.Get("name").MustString("")
			abs_mini := int32(abs_info.Get("range").GetIndex(0).MustInt())
			abs_max := int32(abs_info.Get("range").GetIndex(1).MustInt())
			formatted_value := float64(event.Value-abs_mini) / float64(abs_max-abs_mini)
			_last_value, _ := self.abs_last.Load(name)
			last_value := _last_value.(float64)
			if name == "HAT0X" || name == "HAT0Y" {
				down_up_key := fmt.Sprintf("%s_%s", strconv.FormatFloat(last_value, 'f', 1, 64), strconv.FormatFloat(formatted_value, 'f', 1, 64))
				self.abs_last.Store(name, formatted_value)
				direction := HAT_D_U[down_up_key][0]
				up_down := HAT_D_U[down_up_key][1]
				translated_name := HAT0_KEY_NAME[name][direction]
				self.handel_key_up_down(translated_name, up_down, dev_name)
			} else if name == "LT" || name == "RT" {
				for i := 0; i < 6; i++ {
					if last_value < float64(i)/5 && formatted_value >= float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down("BTN_"+translated_name, DOWN, dev_name)
						if i == 1 {
							self.handel_key_up_down("BTN_"+name, DOWN, dev_name)
						}
					} else if last_value >= float64(i)/5 && formatted_value < float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down("BTN_"+translated_name, UP, dev_name)
						if i == 1 {
							self.handel_key_up_down("BTN_"+name, UP, dev_name)
						}
					}
				}
				self.abs_last.Store(name, formatted_value)
			} else { //必定摇杆
				if self.using_joystick_name != dev_name {
					self.using_joystick_name = dev_name
				}
				// self.abs_last_set(name, formatted_value)
				self.abs_last.Store(name, formatted_value)
				//右摇杆控制视角 只需修改值 有单独线程去处理
				//左摇杆控制轮盘 且与WASD可同时工作 在这里处理
				if (name == "LS_X" || name == "LS_Y") && self.map_on {
					ls_x, ls_y := self.getStick("LS")
					if ls_x == 0.5 && ls_y == 0.5 {
						if self.ls_wheel_released == false {
							self.ls_wheel_released = true
						}
					} else {
						self.ls_wheel_released = false
						wheel_range := self.wheel_range
						if self.wheel_shift_enable {
							wheel_range = self.wheel_shift_range
						}
						target_x := self.wheel_init_x + int32(float64(wheel_range)*2*(ls_x-0.5)) //注意这里的X和Y是相反的
						target_y := self.wheel_init_y + int32(float64(wheel_range)*2*(ls_y-0.5))

						// [新 V1.1.0] 计算路径抖动
						path_jit_x, path_jit_y := self.get_jitter_offset_by_frequency(
							self.path_jitter_amount_px,
							self.path_jitter_wait_frames,
							&self.path_jitter_counter,
							&self.path_jitter_last_x,
							&self.path_jitter_last_y,
						)

						self.handel_wheel_action(Wheel_action_move, target_x+path_jit_x, target_y+path_jit_y)
					}
				}
			}
		} else {
			logger.Warnf("%v config not found", dev_name)
		}
	}
}

func (self *TouchHandler) mix_touch(touch_events chan *event_pack, max_mt_x, max_mt_y int32) {
	wm_size_x, wm_size_y := get_wm_size()
	logger.Infof("xy_wmsize:(%d,%d)", wm_size_x, wm_size_y)
	id_2_vid := make([]int32, 10) //硬件ID到虚拟ID的映射
	var last_id int32 = 0
	pos_s := make([][]int32, 10)
	for i := 0; i < 10; i++ {
		pos_s[i] = make([]int32, 2)
	}
	id_statuses := make([]bool, 10)
	for i := 0; i < 10; i++ {
		id_statuses[i] = false
	}

	translate_xy := func(x, y int32) (int32, int32) { //根据设备方向 将eventX的坐标系转换为标准坐标系
		switch global_device_orientation { //
		case 0: //normal
			return x, y
		case 1: //left side down
			// return y, self.screen_y - x
			return y, wm_size_x - x
		case 2: //up side down
			// return self.screen_y - x, self.screen_x - y
			return wm_size_x - x, wm_size_y - y
		case 3: //right side down
			return wm_size_y - y, x
		default:
			return x, y
		}
	}

	for {
		copy_pos_s := make([][]int32, 10)
		copy(copy_pos_s, pos_s)
		copy_id_statuses := make([]bool, 10)
		copy(copy_id_statuses, id_statuses)
		select {
		case <-global_close_signal:
			return
		case event_pack := <-touch_events:
			for _, event := range event_pack.events {
				switch event.Code {
				case ABS_MT_POSITION_X:
					pos_s[last_id] = []int32{event.Value * wm_size_x / max_mt_x, pos_s[last_id][1]}
				case ABS_MT_POSITION_Y:
					pos_s[last_id] = []int32{pos_s[last_id][0], event.Value * wm_size_y / max_mt_y}
				case ABS_MT_TRACKING_ID:
					if event.Value == -1 {
						id_statuses[last_id] = false
					} else {
						id_statuses[last_id] = true
					}
				case ABS_MT_SLOT:
					last_id = event.Value
				}
			}
			for i := 0; i < 10; i++ {
				if copy_id_statuses[i] != id_statuses[i] {
					if id_statuses[i] { //false -> true 申请
						x, y := translate_xy(pos_s[i][0], pos_s[i][1])
						id_2_vid[i] = self.touch_require(x, y, touch_pos_scale)
						logger.Debugf("mixTouch\trequire\t[%d] translate_xy(%d,%d) => (%d,%d)", i, pos_s[i][0], pos_s[i][1], x, y)
					} else {
						self.touch_release(id_2_vid[i])
						logger.Debugf("mixTouch\trelease\t[%d] ", i)
					}
				} else {
					if pos_s[i][0] != copy_pos_s[i][0] || pos_s[i][1] != copy_pos_s[i][1] {
						x, y := translate_xy(pos_s[i][0], pos_s[i][1])
						self.touch_move(id_2_vid[i], x, y, touch_pos_scale)
						logger.Debugf("mixTouch\tmove\t[%d] translate_xy(%d,%d) => (%d,%d)", i, pos_s[i][0], pos_s[i][1], x, y)
					}
				}
			}

		}
	}
}

func (self *TouchHandler) handel_event() {
	for {
		key_events := make([]*evdev.Event, 0)
		abs_events := make([]*evdev.Event, 0)
		var x int32 = 0
		var y int32 = 0
		var HWhell int32 = 0
		var Wheel int32 = 0
		select {
		case <-global_close_signal:
			return
		case event_pack := <-self.events:
			for _, event := range event_pack.events {
				switch event.Type {
				case evdev.EventKey:
					key_events = append(key_events, event)
				case evdev.EventAbsolute:
					abs_events = append(abs_events, event)
				case evdev.EventRelative:
					switch event.Code {
					case uint16(evdev.RelativeX):
						x = event.Value
					case uint16(evdev.RelativeY):
						y = event.Value
					case uint16(evdev.RelativeHWheel):
						HWhell = event.Value
					case uint16(evdev.RelativeWheel):
						Wheel = event.Value
					}
				}
			}
			var perfPoint time.Time

			perfPoint = time.Now()
			self.handel_rel_event(x, y, HWhell, Wheel)
			rel_sin := time.Since(perfPoint)

			perfPoint = time.Now()
			self.handel_key_events(key_events, event_pack.dev_type, event_pack.dev_name)
			key_sin := time.Since(perfPoint)

			perfPoint = time.Now()
			self.handel_abs_events(abs_events, event_pack.dev_type, event_pack.dev_name)
			abs_sin := time.Since(perfPoint)

			// logger.Debugf("event pack:%v", event_pack)
			logger.Debugf("rel_event\t%v \n", rel_sin)
			logger.Debugf("key_events\t%v \n", key_sin)
			logger.Debugf("abs_events\t%v \n", abs_sin)

		}
	}
}