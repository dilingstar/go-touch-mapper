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
	events             chan *event_pack            //接收事件的channel
	touch_control_func touch_control_func          //发送触屏控制信号的channel
	u_input            chan *u_input_control_pack  //发送u_input控制信号的channel
	map_on             bool                        //映射模式开关
	view_id            int32                       //视角的触摸ID
	wheel_id           int32                       //左摇杆的触摸ID
	allocated_id       []bool                      //10个触摸点分配情况
	config             *simplejson.Json            //映射配置文件
	joystickInfo       map[string]*simplejson.Json //所有摇杆配置文件 dev_name 为key
	screen_x           int32                       //屏幕宽度 (已缩放)
	screen_y           int32                       //屏幕高度 (已缩放)
	rel_screen_x       int32                       //屏幕宽度 (未缩放)
	rel_screen_y       int32                       //屏幕高度 (未缩放)
	view_init_x        int32                       //初始化视角映射的x坐标 (已缩放)
	view_init_y        int32                       //初始化视角映射的y坐标 (已缩放)
	view_current_x     int32                       //当前视角映射的x坐标 (已缩放)
	view_current_y     int32                       //当前视角映射的y坐标 (已缩放)
	view_speed_x       int32                       //视角x方向的速度 (已缩放)
	view_speed_y       int32                       //视角y方向的速度 (已缩放)
	rs_speed_x         float64
	rs_speed_y         float64
	wheel_init_x       int32 //初始化左摇杆映射的x坐标 (未缩放)
	wheel_init_y       int32 //初始化左摇杆映射的y坐标 (未缩放)
	wheel_range        int32 //左摇杆的x轴范围 (未缩放)
	wheel_wasd         []string
	view_lock          sync.Mutex //视角控制相关的锁 用于自动释放和控制相关
	wheel_lock         sync.Mutex //左摇杆控制相关的锁 用于自动释放和控制相关
	touch_control_lock sync.Mutex

	// --- [修改 V1.3.0] auto_release_view_count 重命名并移到 MOUSE 配置中 ---
	auto_release_view_counter int32 // 自动释放计时器 (内部计数)
	// --- [修改 V1.3.0] 结束 ---

	abs_last                sync.Map //abs值的上一次值 用于手柄
	using_joystick_name     string   //当前正在使用的手柄 针对不同手柄死区不同 但程序支持同时插入多个手柄 因此会识别最进发送事件的手柄作为死区配置
	ls_wheel_released       bool     //左摇杆滚轮释放
	wasd_wheel_released     bool     //wasd滚轮释放 两个都释放时 轮盘才会释放
	wasd_wheel_last_x       int32    //wasd滚WHEEL上一次的x坐标
	wasd_wheel_last_y       int32    //wasd滚WHEEL上一次的y坐标
	wasd_up_down_statues    []bool
	key_action_state_save   sync.Map
	BTN_SELECT_UP_DOWN      int32
	KEYBOARD_SWITCH_KEY_NAME_S map[string]bool //键盘切换映射的按键集合
	view_range_limited         bool            //视角是否有界
	map_switch_signal          chan bool
	measure_sensitivity_mode   bool  //计算模式
	total_move_x               int32 //视角总移动距离x
	total_move_y               int32 //视角总移动距离y
	wheel_shift_enable         bool  //启用shift轮盘
	wheel_shift_range          int32

	// --- [修改 V1.1.1] ---
	key_jitter_enable    bool  // [新] 按键抖动开关
	key_jitter_amount_px int32 // [新] 按键抖动幅度(像素) (未缩放)
	// --- [修改 V1.1.1] 结束 ---

	// --- [修改 V1.1.0] ---
	wheel_step_speed float64 // [修改 V1.0.0] 轮盘移动平滑速度 (原版 60)

	wheel_planet_enable    bool    // [新] 行星转圈开关
	wheel_planet_radius_px int32   // [新] 行星转圈半径(像素)
	wheel_planet_speed     float64 // [新] 行星转圈速度 (弧度/帧)
	planet_angle           float64 // [新] 行星当前角度

	wheel_star_x int32 // [新 V1.0.0] 轮盘恒星X
	wheel_star_y int32 // [新 V1.0.0] 轮盘恒星Y
	// --- [修改 V1.1.0] 结束 ---

	// --- [修改 V1.2.3] 彻底重构为 "Curve" (曲线) ---
	shift_press_toggle   bool // [新 V1.2.0] Shift 摁下切换
	shift_release_toggle bool // [新 V1.2.0] Shift 抬起切换

	// 恒星动态速度 (V1.2.3 重命名)
	star_dynamic_speed_enable  bool    // [新 V1.2.0]
	star_dynamic_speed_min     float64 // [新 V1.2.0]
	star_dynamic_speed_freq    float64 // [新 V1.2.0]
	star_dynamic_speed_counter float64 // [新 V1.2.0]

	// 行星动态速度
	planet_dynamic_speed_enable  bool    // [新 V1.2.0]
	planet_dynamic_speed_min     float64 // [新 V1.2.0]
	planet_dynamic_speed_freq    float64 // [新 V1.2.0]
	planet_dynamic_speed_counter float64 // [新 V1.2.0]

	// 独立随机落点
	random_start_enable    bool  // [新 V1.2.1]
	random_start_radius_px int32 // [新 V1.2.1]

	// 恒星曲线 (V1.2.3 新)
	star_curve_enable    bool    // [新 V1.2.3]
	star_curve_amount_px int32   // [新 V1.2.3]
	star_curve_freq      float64 // [新 V1.2.3] (rad/frame)
	star_curve_counter   float64 // [新 V1.2.3] (rad)

	// 行星曲线 (V1.2.3 新)
	planet_curve_enable    bool    // [新 V1.2.3]
	planet_curve_amount_px int32   // [新 V1.2.3]
	planet_curve_freq      float64 // [新 V1.2.3] (V1.2.4 改为乘数)
	// --- [V1.2.3] 结束 ---

	// --- [修改 V1.3.5] P5: 视角 (MOUSE) 配置 ---
	view_auto_release_enable    bool  // (来自配置) 自动释放开关
	view_auto_release_ms        int   // (来自配置) 自动释放时间
	view_reset_radius_enable    bool  // (来自配置) 启用半径重置
	view_reset_radius_px        int32 // (来自配置) 重置半径 (未缩放)
	view_reset_radius_thickness_px int32 // (来自配置) 重置半径厚度 (未缩放)
	view_random_reset_enable    bool  // (来自配置) 启用随机重置
	view_random_reset_radius_px int32 // (来自配置) 随机重置半径 (未缩放)
	// view_saved_x                 int32 // (状态) 用于 SYNC_VIEW_RESET // --- [V1.3.4] 弃用 ---
	// view_saved_y                 int32 // (状态) 用于 SYNC_VIEW_RESET // --- [V1.3.4] 弃用 ---
	// view_is_saved                bool  // (状态) 用于 SYNC_VIEW_RESET // --- [V1.3.4] 弃用, 易导致冲突 ---
	// --- [修改 V1.3.5] 视角 (MOUSE) 结束 ---

	// --- [新增 V1.3.0] 滚轮滑块 (SCROLL_SLIDER) 配置 ---
	scroll_slider_enable           bool    // (来自配置)
	scroll_slider_init_x           int32   // (来自配置) 中心X (未缩放)
	scroll_slider_init_y           int32   // (来自配置) 中心Y (未缩放)
	scroll_slider_bound_up         int32   // (来自配置) 上边界Y (未缩放)
	scroll_slider_bound_down       int32   // (来自配置) 下边界Y (未缩放)
	scroll_slider_timeout_duration time.Duration // (来自配置) 超时
	scroll_slider_speed_px         int32   // (来自配置) 速度 (未缩放)
	scroll_slider_random_enable    bool    // (来自配置)
	scroll_slider_random_radius_px int32   // (来自配置) 随机半径 (未缩放)
	scroll_slider_curve_enable     bool    // (来自配置)
	scroll_slider_curve_amount_px  int32   // (来自配置) 曲线幅度 (未缩放)
	// --- [新增 V1.3.0] 滚轮滑块 (SCROLL_SLIDER) 状态 ---
	scroll_slider_id               int32     // (状态) 触点ID
	scroll_slider_current_y        int32     // (状态) 当前Y (未缩放)
	scroll_slider_last_scroll_time time.Time // (状态) 用于自动释放触点
	scroll_slider_last_reset_time  time.Time // (状态) 用于超时重置位置
	scroll_slider_lock             sync.Mutex
	// --- [新增 V1.3.0] 滚轮滑块结束 ---

	// --- [新增 V1.3.5] P7: 粘滞按键修复 ---
	real_key_down_state sync.Map // 跟踪物理(uinput)按键状态
	// --- [新增 V1.3.5] 结束 ---
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

// [修改 V1.1.0] 基础抖动函数 (保持不变, 仅用于按键)
func get_jitter_offset(amount int32) int32 {
	if amount <= 0 {
		return 0
	}
	return (rand.Int31n(2*amount + 1)) - amount
}

// --- [新增 V1.3.0] 随机偏移辅助函数 ---
// (用于按键抖动, 视角随机重置, 滚轮滑块随机落点)
// (返回 未缩放 坐标)
func (self *TouchHandler) get_random_offset(radius_px int32) (int32, int32) {
	if radius_px <= 0 {
		return 0, 0
	}
	rand_angle := rand.Float64() * 2 * math.Pi
	// 在圆内均匀分布
	rand_radius := rand.Float64() * float64(radius_px)
	offset_x := int32(rand_radius * math.Cos(rand_angle))
	offset_y := int32(rand_radius * math.Sin(rand_angle))
	return offset_x, offset_y
}

// --- [新增 V1.3.0] 结束 ---

// [V1.2.3 新增] 恒星曲线 (S形波浪) 算法 (保留)
func (self *TouchHandler) get_star_curve_offset() (int32, int32) {
	if !self.star_curve_enable || self.star_curve_amount_px == 0 {
		return 0, 0
	}

	// 1. 增加计数器 (弧度)
	self.star_curve_counter += self.star_curve_freq
	if self.star_curve_counter > (2 * math.Pi) {
		self.star_curve_counter -= (2 * math.Pi)
	}

	// 2. 使用 Sin 和 Cos (以不同频率) 来创建平滑的、非圆形的 "蠕动"
	offset_x := math.Sin(self.star_curve_counter*0.7) * float64(self.star_curve_amount_px)
	offset_y := math.Cos(self.star_curve_counter*1.1) * float64(self.star_curve_amount_px)

	return int32(offset_x), int32(offset_y)
}

// [V1.2.3-Final 移除] 移除 V1.2.3 中错误的 get_planet_curve_offset

// --- [新 V1.2.0] 动态速度 (脉动) 函数 ---
// 返回一个在 [min_speed, max_speed] 之间按正弦波变化的值
func (self *TouchHandler) get_dynamic_speed(
	max_speed float64, // 绑定的最快速度
	min_speed float64, // 调节的最慢速度
	freq float64, // 周期频率 (Hz)
	counter *float64, // 计数器 (弧度)
) float64 {
	if min_speed >= max_speed {
		return max_speed // 如果最小值不小于最大值，则返回最大值
	}

	// 1. 计算每帧增加的弧度
	rad_per_frame := (freq * 2 * math.Pi) / MAIN_LOOP_HZ
	*counter += rad_per_frame
	if *counter > (2 * math.Pi) {
		*counter -= (2 * math.Pi)
	}

	// 2. 计算 Sin 值
	sin_wave := (math.Sin(*counter) + 1) / 2.0 // [0, 1]

	// 3. 将 [0, 1] 映射到 [min_speed, max_speed]
	speed_range := max_speed - min_speed
	current_speed := (sin_wave * speed_range) + min_speed

	return current_speed
}

// --- [新 V1.1.1] 按键抖动应用函数 ---
// [修改 V1.3.0] 使用新的 get_random_offset 辅助函数
func (self *TouchHandler) apply_key_jitter(x int32, y int32) (int32, int32) {
	if !self.key_jitter_enable || self.key_jitter_amount_px == 0 {
		return x, y
	}
	// 按键抖动不需要频率，每次都计算
	offset_x, offset_y := self.get_random_offset(self.key_jitter_amount_px)
	return x + offset_x, y + offset_y
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
			content, _ := ioutil.ReadFile(filepath.Join(joystickInfosDir, file.Name()))
			info, _ := simplejson.NewJson(content)
			joystickInfo[file.Name()[:len(file.Name())-5]] = info
			logger.Infof("手柄配置文件已载入 : %s", file.Name())
		}
	}

	abs_last_map := sync.Map{}

	abs_last_map.Store("HAT0X", 0.5)
	abs_last_map.Store("HAT0Y", 0.5)
	abs_last_map.Store("LT", 0.0)
	abs_last_map.Store("RT", 0.0)
	abs_last_map.Store("LS_X", 0.5)
	abs_last_map.Store("LS_Y", 0.5)
	abs_last_map.Store("RS_X", 0.5)
	abs_last_map.Store("RS_Y", 0.5)

	screenSizeX := config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt(3200)
	screenSizeY := config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt(1440)
	// [V1.3.0] 转换为 float64 供后续计算
	screenSizeX_f := float64(screenSizeX)
	screenSizeY_f := float64(screenSizeY)

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
		key_jitter_amount_px_val = int32(key_jitter_amount_val * screenSizeX_f) // (未缩放)
	} else {
		key_jitter_enable_val = true
		key_jitter_amount_px_val = int32(0.003 * screenSizeX_f) // (未缩放)
	}
	// --- [修改 V1.1.2] 结束 ---

	// --- [修改 V1.2.3] 加载轮盘高级配置 (V1.2.3 结构) ---
	wheel_step_speed_val := config_json.Get("WHEEL").Get("STEP_SPEED").MustFloat64(60)

	// 恒星动态速度 (V1.2.3 重命名)
	star_dynamic_speed_enable_val := config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("ENABLE").MustBool(false)
	star_dynamic_speed_min_val := config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("MIN_SPEED").MustFloat64(10.0)
	star_dynamic_speed_freq_val := config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("FREQUENCY").MustFloat64(1.0)

	// 独立随机落点
	random_start_enable_val := config_json.Get("WHEEL").Get("RANDOM_START").Get("ENABLE").MustBool(false)
	random_start_radius_val := config_json.Get("WHEEL").Get("RANDOM_START").Get("RADIUS").MustFloat64(0.01)
	random_start_radius_px_val := int32(random_start_radius_val * screenSizeX_f) // (未缩放)

	// 行星
	wheel_planet_enable_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("ENABLE").MustBool(false)
	wheel_planet_radius_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RADIUS").MustFloat64(0.015)
	wheel_planet_radius_px_val := int32(wheel_planet_radius_val * screenSizeX_f) // (未缩放)
	wheel_planet_speed_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("SPEED").MustFloat64(1.5)

	// 行星动态速度
	planet_dynamic_speed_enable_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("ENABLE").MustBool(false)
	planet_dynamic_speed_min_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("MIN_SPEED").MustFloat64(0.5)
	planet_dynamic_speed_freq_val := config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("FREQUENCY").MustFloat64(1.0)

	// 行星曲线 (V1.2.3 新)
	planet_curve_enable_val := config_json.Get("WHEEL").Get("PLANET_CURVE").Get("ENABLE").MustBool(false)
	planet_curve_amount_val := config_json.Get("WHEEL").Get("PLANET_CURVE").Get("CURVE_AMOUNT").MustFloat64(0.005)
	planet_curve_amount_px_val := int32(planet_curve_amount_val * screenSizeX_f) // (未缩放)
	planet_curve_freq_val := config_json.Get("WHEEL").Get("PLANET_CURVE").Get("CURVE_FREQUENCY").MustFloat64(1.0)
	// [V1.2.4] freq 在 V1.2.4 中是乘数

	// 恒星曲线 (V1.2.3 新)
	star_curve_enable_val := config_json.Get("WHEEL").Get("STAR_CURVE").Get("ENABLE").MustBool(false)
	star_curve_amount_val := config_json.Get("WHEEL").Get("STAR_CURVE").Get("CURVE_AMOUNT").MustFloat64(0.002)
	star_curve_amount_px_val := int32(star_curve_amount_val * screenSizeX_f) // (未缩放)
	star_curve_freq_val := config_json.Get("WHEEL").Get("STAR_CURVE").Get("CURVE_FREQUENCY").MustFloat64(1.0)
	star_curve_freq_rad_val := (star_curve_freq_val * 2 * math.Pi) / MAIN_LOOP_HZ // 转换为 弧度/帧

	// --- [修改 V1.2.3] 加载结束 ---

	// --- [修改 V1.3.5] P2, P5: 加载 MOUSE 视角新配置 ---
	mouse_cfg := config_json.Get("MOUSE")
	view_auto_release_enable_val := mouse_cfg.Get("VIEW_AUTO_RELEASE_ENABLE").MustBool(false) // P5
	view_auto_release_ms_val := mouse_cfg.Get("VIEW_AUTO_RELEASE_MS").MustInt(200)             // P5
	view_reset_radius_enable_val := mouse_cfg.Get("VIEW_RESET_RADIUS_ENABLE").MustBool(false)
	view_reset_radius_px_val := int32(mouse_cfg.Get("VIEW_RESET_RADIUS").MustFloat64(0.1) * screenSizeX_f) // (未缩放)
	view_reset_radius_thickness_px_val := int32(mouse_cfg.Get("VIEW_RESET_RADIUS_THICKNESS").MustFloat64(0.005) * screenSizeX_f) // P2 (未缩放)
	view_random_reset_enable_val := mouse_cfg.Get("VIEW_RANDOM_RESET_ENABLE").MustBool(false)
	view_random_reset_radius_px_val := int32(mouse_cfg.Get("VIEW_RANDOM_RESET_RADIUS").MustFloat64(0.01) * screenSizeX_f) // (未缩放)
	// --- [修改 V1.3.5] MOUSE 加载结束 ---

	// --- [新增 V1.3.0] 加载 SCROLL_SLIDER 滚轮滑块配置 ---
	scroll_cfg := config_json.Get("SCROLL_SLIDER")
	scroll_slider_enable_val := scroll_cfg.Get("ENABLE").MustBool(false)
	scroll_slider_init_x_val := int32(scroll_cfg.Get("POS").GetIndex(0).MustFloat64(0.9) * screenSizeX_f)    // (未缩放)
	scroll_slider_init_y_val := int32(scroll_cfg.Get("POS").GetIndex(1).MustFloat64(0.5) * screenSizeY_f)    // (未缩放)
	scroll_slider_bound_up_val := scroll_slider_init_y_val - int32(scroll_cfg.Get("LENGTH_UP").MustFloat64(0.2)*screenSizeY_f)    // (未缩放)
	scroll_slider_bound_down_val := scroll_slider_init_y_val + int32(scroll_cfg.Get("LENGTH_DOWN").MustFloat64(0.2)*screenSizeY_f) // (未缩放)
	scroll_slider_timeout_duration_val := time.Duration(scroll_cfg.Get("TIMEOUT_S").MustFloat64(3.0) * float64(time.Second))
	// (未缩放) 速度 1.0 = 1% 屏幕高度
	scroll_slider_speed_px_val := int32(scroll_cfg.Get("SPEED").MustFloat64(1.0) * 0.01 * screenSizeY_f)
	scroll_slider_random_enable_val := scroll_cfg.Get("RANDOM_START_ENABLE").MustBool(false)
	scroll_slider_random_radius_px_val := int32(scroll_cfg.Get("RANDOM_START_RADIUS").MustFloat64(0.005) * screenSizeX_f) // (未缩放)
	scroll_slider_curve_enable_val := scroll_cfg.Get("CURVE_ENABLE").MustBool(false)
	scroll_slider_curve_amount_px_val := int32(scroll_cfg.Get("CURVE_AMOUNT").MustFloat64(0.005) * screenSizeX_f) // (未缩放)
	// --- [新增 V1.3.0] SCROLL_SLIDER 加载结束 ---

	return &TouchHandler{
		events:             events,
		touch_control_func: touch_control_func,
		u_input:            u_input,
		map_on:             false, //false
		view_id:            -1,
		wheel_id:           -1,
		allocated_id:       make([]bool, 12),
		config:             config_json,
		joystickInfo:       joystickInfo,
		screen_x:           int32(screenSizeX << touch_pos_scale), // (已缩放)
		screen_y:           int32(screenSizeY << touch_pos_scale), // (已缩放)
		rel_screen_x:       int32(screenSizeX),                    // (未缩放)
		rel_screen_y:       int32(screenSizeY),                    // (未缩放)
		view_init_x:        int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)), // (已缩放)
		view_init_y:        int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)), // (已缩放)
		view_current_x:     int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)), // (已缩放)
		view_current_y:     int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)), // (已缩放)
		view_speed_x:       int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(0).MustFloat64()), // (已缩放)
		view_speed_y:       int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(1).MustFloat64()), // (已缩放)
		rs_speed_x:         32,
		rs_speed_y:         32,
		wheel_init_x:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * screenSizeX_f), // (未缩放)
		wheel_init_y:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * screenSizeY_f), // (未缩放)
		wheel_range:        int32(config_json.Get("WHEEL").Get("RANGE").MustFloat64() * screenSizeX_f),           // (未缩放)
		wheel_wasd: []string{
			config_json.Get("WHEEL").Get("WASD").GetIndex(0).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(1).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(2).MustString(),
			config_json.Get("WHEEL").Get("WASD").GetIndex(3).MustString(),
		},
		view_lock:                 sync.Mutex{},
		wheel_lock:                sync.Mutex{},
		touch_control_lock:        sync.Mutex{},
		auto_release_view_counter: 0, // [V1.3.0] 重命名
		abs_last:                  abs_last_map,
		using_joystick_name:       "",
		ls_wheel_released:         true,
		wasd_wheel_released:       true,
		// [V1.2.5 回滚] 保持 V1.2.3/V1.1.0 逻辑
		wasd_wheel_last_x:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * screenSizeX_f),
		wasd_wheel_last_y:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * screenSizeY_f),
		wasd_up_down_statues:    make([]bool, 5), //放置wasd的状态与shift启用下，shift的状态
		key_action_state_save:   sync.Map{},
		BTN_SELECT_UP_DOWN:      0,
		KEYBOARD_SWITCH_KEY_NAME_S: KEYBOARD_SWITCH_KEY_NAME_S,
		view_range_limited:         view_range_limited,
		map_switch_signal:          map_switch_signal,
		measure_sensitivity_mode:   measure_sensitivity_mode,
		wheel_shift_enable:         config_json.Get("WHEEL").Get("SHIFT_RANGE_ENABLE").MustBool(false),
		wheel_shift_range: int32(config_json.Get("WHEEL").Get("SHIFT_RANGE").MustFloat64() * screenSizeX_f), // (未缩放)

		// --- [修改 V1.1.2] 初始化 KEY_JITTER ---
		key_jitter_enable:    key_jitter_enable_val,
		key_jitter_amount_px: key_jitter_amount_px_val, // (未缩放)
		// --- [修改 V1.1.2] 结束 ---

		// --- [修改 V1.2.3] 初始化轮盘高级配置 ---
		wheel_step_speed:    wheel_step_speed_val,

		// 恒星动态速度
		star_dynamic_speed_enable:  star_dynamic_speed_enable_val,
		star_dynamic_speed_min:     star_dynamic_speed_min_val,
		star_dynamic_speed_freq:    star_dynamic_speed_freq_val,
		star_dynamic_speed_counter: 0,

		// 独立随机落点
		random_start_enable:    random_start_enable_val,
		random_start_radius_px: random_start_radius_px_val, // (未缩放)

		// 行星
		wheel_planet_enable:    wheel_planet_enable_val,
		wheel_planet_radius_px: wheel_planet_radius_px_val, // (未缩放)
		wheel_planet_speed:     wheel_planet_speed_val / MAIN_LOOP_HZ, // 转换为 弧度/帧
		planet_angle:           0,

		// 行星动态速度
		planet_dynamic_speed_enable:  planet_dynamic_speed_enable_val,
		planet_dynamic_speed_min:     planet_dynamic_speed_min_val,
		planet_dynamic_speed_freq:    planet_dynamic_speed_freq_val,
		planet_dynamic_speed_counter: 0,

		// 行星曲线 (V1.2.3 新)
		planet_curve_enable:    planet_curve_enable_val,
		planet_curve_amount_px: planet_curve_amount_px_val, // (未缩放)
		planet_curve_freq:      planet_curve_freq_val, // [V1.2.4] freq 是乘数

		// 恒星曲线 (V1.2.3 新)
		star_curve_enable:    star_curve_enable_val,
		star_curve_amount_px: star_curve_amount_px_val, // (未缩放)
		star_curve_freq:      star_curve_freq_rad_val, // (rad/frame)
		star_curve_counter:   0,

		// Shift 逻辑
		shift_press_toggle:   config_json.Get("WHEEL").Get("SHIFT_PRESS_TOGGLE").MustBool(false),
		shift_release_toggle: config_json.Get("WHEEL").Get("SHIFT_RELEASE_TOGGLE").MustBool(false),

		wheel_star_x: 0,
		wheel_star_y: 0,
		// --- [修改 V1.2.3] 初始化结束 ---

		// --- [修改 V1.3.5] P2, P5: 初始化 MOUSE 视角新配置 ---
		view_auto_release_enable:    view_auto_release_enable_val,
		view_auto_release_ms:        view_auto_release_ms_val,
		view_reset_radius_enable:    view_reset_radius_enable_val,
		view_reset_radius_px:        view_reset_radius_px_val, // (未缩放)
		view_reset_radius_thickness_px: view_reset_radius_thickness_px_val, // P2 (未缩放)
		view_random_reset_enable:    view_random_reset_enable_val,
		view_random_reset_radius_px: view_random_reset_radius_px_val, // (未缩放)
		// view_is_saved:                false, // [V1.3.4] 弃用
		// --- [修改 V1.3.5] MOUSE 初始化结束 ---

		// --- [新增 V1.3.0] 初始化 SCROLL_SLIDER 滚轮滑块配置 ---
		scroll_slider_enable:           scroll_slider_enable_val,
		scroll_slider_init_x:           scroll_slider_init_x_val,   // (未缩放)
		scroll_slider_init_y:           scroll_slider_init_y_val,   // (未缩放)
		scroll_slider_bound_up:         scroll_slider_bound_up_val, // (未缩放)
		scroll_slider_bound_down:       scroll_slider_bound_down_val, // (未缩放)
		scroll_slider_timeout_duration: scroll_slider_timeout_duration_val,
		scroll_slider_speed_px:         scroll_slider_speed_px_val, // (未缩放)
		scroll_slider_random_enable:    scroll_slider_random_enable_val,
		scroll_slider_random_radius_px: scroll_slider_random_radius_px_val, // (未缩放)
		scroll_slider_curve_enable:     scroll_slider_curve_enable_val,
		scroll_slider_curve_amount_px:  scroll_slider_curve_amount_px_val, // (未缩放)
		// --- [新增 V1.3.0] SCROLL_SLIDER 状态 ---
		scroll_slider_id:               -1,
		scroll_slider_current_y:        scroll_slider_init_y_val, // (未缩放)
		scroll_slider_last_scroll_time: time.Now(),
		scroll_slider_last_reset_time:  time.Now(),
		scroll_slider_lock:             sync.Mutex{},
		// --- [新增 V1.3.0] SCROLL_SLIDER 初始化结束 ---

		// --- [新增 V1.3.5] P7: 初始化 粘滞按键 映射 ---
		real_key_down_state: sync.Map{},
		// --- [新增 V1.3.5] 结束 ---
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
	screenSizeX := config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt(3200)
	screenSizeY := config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt(1440)
	// [V1.3.0] 转换为 float64 供后续计算
	screenSizeX_f := float64(screenSizeX)
	screenSizeY_f := float64(screenSizeY)

	self.config = config_json
	self.screen_x = int32(screenSizeX << touch_pos_scale) // (已缩放)
	self.screen_y = int32(screenSizeY << touch_pos_scale) // (已缩放)
	self.rel_screen_x = int32(screenSizeX)                // (未缩放)
	self.rel_screen_y = int32(screenSizeY)                // (未缩放)
	self.view_init_x = int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)) // (已缩放)
	self.view_init_y = int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)) // (已缩放)
	self.view_current_x = int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustFloat64() * float64(screenSizeX<<touch_pos_scale)) // (已缩放)
	self.view_current_y = int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustFloat64() * float64(screenSizeY<<touch_pos_scale)) // (已缩放)
	self.view_speed_x = int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(0).MustFloat64()) // (已缩放)
	self.view_speed_y = int32((1 << touch_pos_scale) * config_json.Get("MOUSE").Get("SPEED").GetIndex(1).MustFloat64()) // (已缩放)
	self.wheel_init_x = int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * screenSizeX_f) // (未缩放)
	self.wheel_init_y = int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * screenSizeY_f) // (未缩放)
	self.wheel_range = int32(config_json.Get("WHEEL").Get("RANGE").MustFloat64() * screenSizeX_f)           // (未缩放)
	self.wheel_wasd = []string{
		config_json.Get("WHEEL").Get("WASD").GetIndex(0).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(1).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(2).MustString(),
		config_json.Get("WHEEL").Get("WASD").GetIndex(3).MustString(),
	}
	self.wasd_wheel_last_x = int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustFloat64() * screenSizeX_f) // (未缩放)
	self.wasd_wheel_last_y = int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustFloat64() * screenSizeY_f) // (未缩放)
	self.KEYBOARD_SWITCH_KEY_NAME_S = make(map[string]bool)
	for _, key := range config_json.Get("MOUSE").Get("SWITCH_KEYS").MustStringArray() {
		if key != "" {
			self.KEYBOARD_SWITCH_KEY_NAME_S[key] = true
		} else {
			logger.Warnf("映射配置文件中有空的键盘切换按键,请检查配置文件")
		}
	}

	self.wheel_shift_enable = config_json.Get("WHEEL").Get("SHIFT_RANGE_ENABLE").MustBool(false)
	self.wheel_shift_range = int32(config_json.Get("WHEEL").Get("SHIFT_RANGE").MustFloat64() * screenSizeX_f) // (未缩放)

	// --- [修改 V1.1.2] 安全加载 KEY_JITTER ---
	if keyJitterJSON, ok := config_json.CheckGet("KEY_JITTER"); ok {
		self.key_jitter_enable = keyJitterJSON.Get("ENABLE").MustBool(true)
		self.key_jitter_amount_px = int32(keyJitterJSON.Get("AMOUNT").MustFloat64(0.003) * screenSizeX_f) // (未缩放)
	} else {
		self.key_jitter_enable = true
		self.key_jitter_amount_px = int32(0.003 * screenSizeX_f) // (未缩放)
	}
	// --- [修改 V1.1.2] 结束 ---

	// --- [修改 V1.2.3] 重新加载轮盘高级配置 ---
	self.wheel_step_speed = config_json.Get("WHEEL").Get("STEP_SPEED").MustFloat64(60)

	// 恒星动态速度
	self.star_dynamic_speed_enable = config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("ENABLE").MustBool(false)
	self.star_dynamic_speed_min = config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("MIN_SPEED").MustFloat64(10.0)
	self.star_dynamic_speed_freq = config_json.Get("WHEEL").Get("STAR_DYNAMIC_SPEED").Get("FREQUENCY").MustFloat64(1.0)

	// 独立随机落点
	self.random_start_enable = config_json.Get("WHEEL").Get("RANDOM_START").Get("ENABLE").MustBool(false)
	self.random_start_radius_px = int32(config_json.Get("WHEEL").Get("RANDOM_START").Get("RADIUS").MustFloat64(0.01) * screenSizeX_f) // (未缩放)

	// 行星
	self.wheel_planet_enable = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("ENABLE").MustBool(false)
	self.wheel_planet_radius_px = int32(config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("RADIUS").MustFloat64(0.015) * screenSizeX_f) // (未缩放)
	self.wheel_planet_speed = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("SPEED").MustFloat64(1.5) / MAIN_LOOP_HZ // 转换为 弧度/帧

	// 行星动态速度
	self.planet_dynamic_speed_enable = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("ENABLE").MustBool(false)
	self.planet_dynamic_speed_min = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("MIN_SPEED").MustFloat64(0.5)
	self.planet_dynamic_speed_freq = config_json.Get("WHEEL").Get("WHEEL_PLANET").Get("PLANET_DYNAMIC_SPEED").Get("FREQUENCY").MustFloat64(1.0)

	// 行星曲线 (V1.2.3 新)
	self.planet_curve_enable = config_json.Get("WHEEL").Get("PLANET_CURVE").Get("ENABLE").MustBool(false)
	self.planet_curve_amount_px = int32(config_json.Get("WHEEL").Get("PLANET_CURVE").Get("CURVE_AMOUNT").MustFloat64(0.005) * screenSizeX_f) // (未缩放)
	planet_curve_freq_val := config_json.Get("WHEEL").Get("PLANET_CURVE").Get("CURVE_FREQUENCY").MustFloat64(1.0)
	self.planet_curve_freq = planet_curve_freq_val // [V1.2.4] freq 是乘数

	// 恒星曲线 (V1.2.3 新)
	self.star_curve_enable = config_json.Get("WHEEL").Get("STAR_CURVE").Get("ENABLE").MustBool(false)
	self.star_curve_amount_px = int32(config_json.Get("WHEEL").Get("STAR_CURVE").Get("CURVE_AMOUNT").MustFloat64(0.002) * screenSizeX_f) // (未缩放)
	star_curve_freq_val := config_json.Get("WHEEL").Get("STAR_CURVE").Get("CURVE_FREQUENCY").MustFloat64(1.0)
	self.star_curve_freq = (star_curve_freq_val * 2 * math.Pi) / MAIN_LOOP_HZ // 转换为 弧度/帧

	// Shift 逻辑
	self.shift_press_toggle = config_json.Get("WHEEL").Get("SHIFT_PRESS_TOGGLE").MustBool(false)
	self.shift_release_toggle = config_json.Get("WHEEL").Get("SHIFT_RELEASE_TOGGLE").MustBool(false)
	// --- [修改 V1.2.3] 重新加载结束 ---

	// --- [修改 V1.3.5] P2, P5: 重新加载 MOUSE 视角新配置 ---
	mouse_cfg := config_json.Get("MOUSE")
	self.view_auto_release_enable = mouse_cfg.Get("VIEW_AUTO_RELEASE_ENABLE").MustBool(false) // P5
	self.view_auto_release_ms = mouse_cfg.Get("VIEW_AUTO_RELEASE_MS").MustInt(200)             // P5
	self.view_reset_radius_enable = mouse_cfg.Get("VIEW_RESET_RADIUS_ENABLE").MustBool(false)
	self.view_reset_radius_px = int32(mouse_cfg.Get("VIEW_RESET_RADIUS").MustFloat64(0.1) * screenSizeX_f) // (未缩放)
	self.view_reset_radius_thickness_px = int32(mouse_cfg.Get("VIEW_RESET_RADIUS_THICKNESS").MustFloat64(0.005) * screenSizeX_f) // P2 (未缩放)
	self.view_random_reset_enable = mouse_cfg.Get("VIEW_RANDOM_RESET_ENABLE").MustBool(false)
	self.view_random_reset_radius_px = int32(mouse_cfg.Get("VIEW_RANDOM_RESET_RADIUS").MustFloat64(0.01) * screenSizeX_f) // (未缩放)
	// self.view_is_saved = false // [V1.3.4] 弃用
	// --- [修改 V1.3.5] MOUSE 重新加载结束 ---

	// --- [新增 V1.3.0] 重新加载 SCROLL_SLIDER 滚轮滑块配置 ---
	scroll_cfg := config_json.Get("SCROLL_SLIDER")
	self.scroll_slider_enable = scroll_cfg.Get("ENABLE").MustBool(false)
	self.scroll_slider_init_x = int32(scroll_cfg.Get("POS").GetIndex(0).MustFloat64(0.9) * screenSizeX_f)    // (未缩放)
	self.scroll_slider_init_y = int32(scroll_cfg.Get("POS").GetIndex(1).MustFloat64(0.5) * screenSizeY_f)    // (未缩放)
	self.scroll_slider_bound_up = self.scroll_slider_init_y - int32(scroll_cfg.Get("LENGTH_UP").MustFloat64(0.2)*screenSizeY_f)    // (未缩放)
	self.scroll_slider_bound_down = self.scroll_slider_init_y + int32(scroll_cfg.Get("LENGTH_DOWN").MustFloat64(0.2)*screenSizeY_f) // (未缩放)
	self.scroll_slider_timeout_duration = time.Duration(scroll_cfg.Get("TIMEOUT_S").MustFloat64(3.0) * float64(time.Second))
	// (未缩放) 速度 1.0 = 1% 屏幕高度
	self.scroll_slider_speed_px = int32(scroll_cfg.Get("SPEED").MustFloat64(1.0) * 0.01 * screenSizeY_f)
	self.scroll_slider_random_enable = scroll_cfg.Get("RANDOM_START_ENABLE").MustBool(false)
	self.scroll_slider_random_radius_px = int32(scroll_cfg.Get("RANDOM_START_RADIUS").MustFloat64(0.005) * screenSizeX_f) // (未缩放)
	self.scroll_slider_curve_enable = scroll_cfg.Get("CURVE_ENABLE").MustBool(false)
	self.scroll_slider_curve_amount_px = int32(scroll_cfg.Get("CURVE_AMOUNT").MustFloat64(0.005) * screenSizeX_f) // (未缩放)
	// --- [新增 V1.3.0] SCROLL_SLIDER 重置状态 ---
	self.scroll_slider_id = -1
	self.scroll_slider_current_y = self.scroll_slider_init_y // (未缩放)
	self.scroll_slider_last_scroll_time = time.Now()
	self.scroll_slider_last_reset_time = time.Now()
	// --- [新增 V1.3.0] SCROLL_SLIDER 重新加载结束 ---
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

// [V1.2.3-Final 修复] 移植 V1.2.5 的“扁平线”和 V1.2.4 的“`planet_angle`驱动”
func (self *TouchHandler) get_planet_pos() (int32, int32) {
	// 1. 获取恒星位置 (恒星位置已在 handel_wheel_action 中包含了 "Star Curve")
	final_star_x := self.wheel_star_x
	final_star_y := self.wheel_star_y

	// 2. 如果行星功能关闭，直接返回恒星位置
	if !self.wheel_planet_enable {
		return final_star_x, final_star_y
	}

	// 3. 计算行星动态速度
	var current_planet_speed float64
	if self.planet_dynamic_speed_enable {
		min_speed_rad_per_frame := (self.planet_dynamic_speed_min / MAIN_LOOP_HZ)
		current_planet_speed = self.get_dynamic_speed(
			self.wheel_planet_speed, // Max speed (rad/frame)
			min_speed_rad_per_frame, // Min speed (rad/frame)
			self.planet_dynamic_speed_freq,
			&self.planet_dynamic_speed_counter,
		)
	} else {
		current_planet_speed = self.wheel_planet_speed
	}

	// 4. 增加行星主角度
	self.planet_angle += current_planet_speed
	if self.planet_angle > 2*math.Pi {
		self.planet_angle -= 2 * math.Pi
	}

	// 5. [V1.2.3-Final] 检查是否启用“行星曲线” (扁平线模式)
	if self.planet_curve_enable && self.planet_curve_amount_px > 0 {
		// --- “扁平线”模式 (来自 V1.2.5) ---
		// [V1.2.4] 使用行星自己的角度 (planet_angle) 乘以频率 (乘数)
		curve_calc_angle := self.planet_angle * self.planet_curve_freq
		curve_amount_float := float64(self.planet_curve_amount_px)

		// [V1.2.5] 曲线(幅度) *成为* 半径, 在 [-幅度, +幅度] 之间波动
		// 使用 Sin 和 Cos 叠加，制造不规则但平滑的波动
		final_radius := (math.Sin(curve_calc_angle*1.7)*0.5 + math.Cos(curve_calc_angle*0.9)*0.5) * curve_amount_float

		// [V1.2.5] 角度 *是* 行星的主角度 (使这条线旋转)
		final_angle := self.planet_angle

		// 7. [V1.2.5] 转换为笛卡尔坐标
		planet_x := final_star_x + int32(final_radius*math.Cos(final_angle))
		planet_y := final_star_y + int32(final_radius*math.Sin(final_angle))
		return planet_x, planet_y

	} else {
		// --- 正常“圆形”模式 ---
		final_radius := float64(self.wheel_planet_radius_px)
		final_angle := self.planet_angle

		// 7. [V1.2.5] 转换为笛卡尔坐标
		planet_x := final_star_x + int32(final_radius*math.Cos(final_angle))
		planet_y := final_star_y + int32(final_radius*math.Sin(final_angle))
		return planet_x, planet_y
	}
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
				if self.wheel_id != -1 {
					// [V1.2.3-Final] get_planet_pos() 已包含最新曲线算法
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

// --- [新增 V1.3.0] 滚轮滑块：自动释放循环 ---
func (self *TouchHandler) loop_auto_release_scroll_slider() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			self.scroll_slider_lock.Lock()
			// 检查是否需要释放 (例如 50ms 未收到新滚动事件)
			if self.scroll_slider_id != -1 {
				if time.Since(self.scroll_slider_last_scroll_time) > time.Millisecond*50 {
					self.scroll_slider_id = self.touch_release(self.scroll_slider_id)
				}
			}
			self.scroll_slider_lock.Unlock()
			time.Sleep(time.Duration(20) * time.Millisecond) // 每 20ms 检查一次
		}
	}
}

// --- [新增 V1.3.0] 结束 ---

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

// --- [新增 V1.3.0] 视角重置辅助函数 ---
// (接收 已缩放 坐标)
func (self *TouchHandler) reset_view_position(new_x int32, new_y int32) {
	final_x, final_y := new_x, new_y

	// 1. 应用随机重置
	if self.view_random_reset_enable {
		// get_random_offset 返回 (未缩放) 坐标
		offset_x, offset_y := self.get_random_offset(self.view_random_reset_radius_px)
		// 转换为 (已缩放) 偏移
		final_x += (offset_x << touch_pos_scale)
		final_y += (offset_y << touch_pos_scale)
	}

	// 2. 更新当前坐标
	self.view_current_x = final_x
	self.view_current_y = final_y

	// 3. 如果触点已按下, 则重置
	if self.view_id != -1 {
		// (使用 scale=0, 因为坐标已缩放)
		tmp_view_id := self.touch_require(self.view_current_x, self.view_current_y, 0)
		// 移动到新位置 (虽然 require 已经在了, 但 move 确保)
		self.touch_move(tmp_view_id, self.view_current_x, self.view_current_y, 0)
		// 释放旧触点
		self.touch_release(self.view_id)
		self.view_id = tmp_view_id
	}
}

// --- [新增 V1.3.0] 结束 ---

// [修改 V1.3.5] P1, P2, P3: 重构视角移动和重置逻辑
func (self *TouchHandler) handel_view_move(offset_x int32, offset_y int32) { //视角移动
	self.view_lock.Lock()
	defer self.view_lock.Unlock()
	if self.measure_sensitivity_mode {
		self.total_move_x += offset_x
		self.total_move_y += offset_y
		logger.Infof("total_move_x:%v\ttotal_move_y:%v", self.total_move_x, self.total_move_y)
	}
	self.auto_release_view_counter = 0 // [V1.3.0] 重命名
	if self.view_id == -1 {
		// 仅在按下时应用随机重置 (如果启用)
		self.reset_view_position(self.view_init_x, self.view_init_y)
		self.view_id = self.touch_require(self.view_current_x, self.view_current_y, 0)
	}
	self.view_current_x += offset_x * self.view_speed_x
	self.view_current_y += offset_y * self.view_speed_y

	if self.view_range_limited { //有界 or 无界 即 使用eventX 还是 inputManager

		var reset_required bool = false

		// --- [修改 V1.3.5] P3: 原版屏幕边缘重置 (始终启用) ---
		if self.view_current_x <= 0 || self.view_current_x >= self.screen_x || self.view_current_y <= 0 || self.view_current_y >= self.screen_y {
			reset_required = true
		}
		// --- [修改 V1.3.5] P3 结束 ---

		// --- [修改 V1.3.5] P1, P2: "圆环"重置 (仅在未被屏幕边缘重置时检查) ---
		// P1 (重置键失效) 被此逻辑自动修复
		if !reset_required && self.view_reset_radius_enable {
			// (坐标已缩放, 半径需要转换为未缩放进行比较)
			dist_x := (self.view_current_x - self.view_init_x) >> touch_pos_scale // (未缩放) 距离
			dist_y := (self.view_current_y - self.view_init_y) >> touch_pos_scale // (未缩放) 距离
			dist_sq := float64(dist_x*dist_x + dist_y*dist_y)

			// (半径和厚度都是未缩放的)
			radius_px := self.view_reset_radius_px
			thickness_px := self.view_reset_radius_thickness_px
			
			// 计算"圆环"的内径和外径
			inner_radius := radius_px - thickness_px
			if inner_radius < 0 {
				inner_radius = 0 // 防止厚度大于半径
			}
			
			inner_radius_sq := float64(inner_radius * inner_radius)
			outer_radius_sq := float64(radius_px * radius_px)
			
			// 检查是否在"圆环"内
			if dist_sq >= inner_radius_sq && dist_sq <= outer_radius_sq {
				reset_required = true
			}
		}
		// --- [修改 V1.3.5] P1, P2 结束 ---

		if reset_required {
			// 使用新的重置函数 (它包含随机逻辑)
			self.reset_view_position(self.view_init_x, self.view_init_y)
			// 在新位置上应用本次移动
			self.view_current_x += offset_x * self.view_speed_x
			self.view_current_y += offset_y * self.view_speed_y
			self.touch_move(self.view_id, self.view_current_x, self.view_current_y, 0)
		} else {
			// 未重置, 正常移动
			self.touch_move(self.view_id, self.view_current_x, self.view_current_y, 0)
		}

	} else {
		self.touch_move(self.view_id, self.view_current_x, self.view_current_y, 0)
	}
}

// [修改 V1.3.5] P4, P5: 修复自动释放配置无效的 BUG
func (self *TouchHandler) auto_handel_view_release() { //视角释放
	for {
		select {
		case <-global_close_signal:
			return
		default:
			// --- [修改 V1.3.5] P4: 将配置读取移入循环, 使其可热重载 ---
			timeout := self.view_auto_release_ms
			enable := self.view_auto_release_enable
			// --- [修改 V1.3.5] P4 结束 ---

			if !enable { // P5: 检查开关
				time.Sleep(time.Duration(200) * time.Millisecond) // 如果禁用, 睡眠更久以减少 CPU 占用
				continue
			}

			if timeout > 0 {
				self.view_lock.Lock()
				if self.view_id != -1 {
					self.auto_release_view_counter += 1 // [V1.3.0] 重命名
					// (50ms 检查一次)
					if self.auto_release_view_counter > int32(timeout/50) {
						self.auto_release_view_counter = 0 // [V1.3.0] 重命名
						self.view_id = self.touch_release(self.view_id)
					}
				}
				self.view_lock.Unlock()
			}
			time.Sleep(time.Duration(50) * time.Millisecond) // 50ms 检查周期
		}
	}
}

// [修改 V1.2.3] 轮盘核心逻辑重构 (修复“直线”BUG 和 实现“随机落点”)
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

		// [V1.2.1] 按下逻辑
		if self.wheel_id == -1 {
			press_x, press_y := abs_x, abs_y

			// [V1.2.1] 检查是否启用“随机落点”
			if self.random_start_enable {
				// [V1.3.0] 使用新的 get_random_offset 辅助函数
				offset_x, offset_y := self.get_random_offset(self.random_start_radius_px)
				press_x = self.wheel_init_x + offset_x
				press_y = self.wheel_init_y + offset_y
			}

			// 1. [V1.2.1] 先设置“恒星”位置 (V1.2.3: 已包含 Star Curve)
			self.wheel_star_x = press_x
			self.wheel_star_y = press_y

			// 2. [V1.2.1] 重置行星角度
			self.planet_angle = 0 // 重置角度

			// 3. [V1.2.1] 立即计算最终的抖动/行星位置
			final_x, final_y := self.get_planet_pos()

			// 4. [V1.2.1] 在最终位置按下触点，修复“直线”BUG
			self.wheel_id = self.touch_require(final_x, final_y, touch_pos_scale)

		} else { // [V1.2.1] 移动逻辑
			// 仅更新“恒星”位置 (V1.2.3: 已包含 Star Curve)
			self.wheel_star_x = abs_x
			self.wheel_star_y = abs_y
		}
	}
}

// [V1.2.5 回滚] 回滚到 V1.2.3/V1.1.0 的8“角”点逻辑 (解决 V1.2.5 的"8边形"BUG)
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
		return self.wheel_init_x + x*wheel_range, self.wheel_init_y + y*wheel_range
	} else {
		// [V1.2.3-Final] 此处使用原版 707/1000 逻辑
		return self.wheel_init_x + x*wheel_range*707/1000, self.wheel_init_y + y*wheel_range*707/1000
	}
}

// [修改 V1.2.0] 使用可配置的 step_speed
func (self *TouchHandler) update_wheel_xy(last_x, last_y, target_x, target_y int32, step_speed float64) (int32, int32) {
	if last_x == target_x && last_y == target_y {
		return last_x, last_y
	} else {
		x_rest := target_x - last_x
		y_rest := target_y - last_y
		total_rest := int32(math.Sqrt(float64(x_rest*x_rest + y_rest*y_rest)))

		// [V1.2.3 修复] 确保 step_speed 至少为 1 (如果 < 1 会导致除零或不移动)
		var wheel_step_val_int32 int32
		if step_speed < 1.0 {
			wheel_step_val_int32 = 1
		} else {
			wheel_step_val_int32 = int32(step_speed)
		}

		if total_rest <= wheel_step_val_int32 {
			return target_x, target_y
		} else {
			// [V1.2.3 修复] 确保 total_rest 不为 0
			if total_rest == 0 {
				return target_x, target_y
			}
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
			wasd_wheel_target_x, wasd_wheel_target_y := self.get_wasd_now_target() // [V1.2.3-Final] (8“角”点)

			if self.wheel_init_x == wasd_wheel_target_x && self.wheel_init_y == wasd_wheel_target_y {
				// [V1.2.3] 恒星停止移动
				self.wasd_wheel_released = true
				self.wasd_wheel_last_x = self.wheel_init_x
				self.wasd_wheel_last_y = self.wheel_init_y
				self.star_curve_counter = 0 // [V1.2.3] 重置恒星曲线计数器
			} else {
				// [V1.2.3] 恒星正在移动
				self.wasd_wheel_released = false
				if self.wasd_wheel_last_x != wasd_wheel_target_x || self.wasd_wheel_last_y != wasd_wheel_target_y {

					// --- [修改 V1.2.3] 计算恒星动态速度 ---
					var current_step_speed float64
					if self.star_dynamic_speed_enable {
						current_step_speed = self.get_dynamic_speed(
							self.wheel_step_speed, // Max speed
							self.star_dynamic_speed_min,
							self.star_dynamic_speed_freq,
							&self.star_dynamic_speed_counter,
						)
					} else {
						current_step_speed = self.wheel_step_speed
					}
					// --- [修改 V1.2.3] 结束 ---

					// 1. [V1.2.3] 计算“主路径”上的下一个点
					main_path_x, main_path_y := self.update_wheel_xy(self.wasd_wheel_last_x, self.wasd_wheel_last_y, wasd_wheel_target_x, wasd_wheel_target_y, current_step_speed)
					self.wasd_wheel_last_x = main_path_x
					self.wasd_wheel_last_y = main_path_y

					// 2. [V1.2.3] 计算“恒星曲线”的偏移量
					curve_x, curve_y := self.get_star_curve_offset()

					// 3. [V1.2.3] 施加曲线
					final_x := main_path_x + curve_x
					final_y := main_path_y + curve_y

					// 4. [V1.2.3] 更新恒星位置
					self.handel_wheel_action(Wheel_action_move, final_x, final_y)
				}
			}
			if self.wheel_id != -1 && self.wasd_wheel_released && self.ls_wheel_released {
				self.handel_wheel_action(Wheel_action_release, -1, -1) //wheel当前按下 且两个标记都释放 则释放
			}
			time.Sleep(time.Duration(MAIN_LOOP_NS_INT) * time.Nanosecond) // 250HZ
		}
	}
}

func (self *TouchHandler) quick_click(keyname string) {
	self.handel_key_up_down(keyname, DOWN, "MOUSE_WHEEL")
	time.Sleep(time.Duration(50) * time.Millisecond)
	self.handel_key_up_down(keyname, UP, "MOUSE_WHEEL")
}

// --- [新增 V1.3.0] 滚轮滑块核心逻辑 ---
func (self *TouchHandler) handel_scroll_slider(direction int32) {
	// (direction: -1 为上, 1 为下)

	// 1. 检查功能是否启用
	if !self.scroll_slider_enable {
		// 回退到原版 quick_click 逻辑
		if direction < 0 {
			go self.quick_click("REL_WHEEL_UP") //纵向滚轮向上
		} else if direction > 0 {
			go self.quick_click("REL_WHEEL_DOWN") //纵向滚轮向下
		}
		return
	}

	self.scroll_slider_lock.Lock()
	defer self.scroll_slider_lock.Unlock()

	// 2. 更新滚动时间 (用于自动释放)
	self.scroll_slider_last_scroll_time = time.Now()

	// 3. 检查是否超时 (用于重置位置)
	if time.Since(self.scroll_slider_last_reset_time) > self.scroll_slider_timeout_duration {
		self.scroll_slider_current_y = self.scroll_slider_init_y
	}
	// 4. 更新重置计时器
	self.scroll_slider_last_reset_time = time.Now()

	// 5. 准备坐标
	final_x, final_y := self.scroll_slider_init_x, self.scroll_slider_current_y

	// 6. 如果是新按下, 应用随机落点
	if self.scroll_slider_id == -1 && self.scroll_slider_random_enable {
		offset_x, offset_y := self.get_random_offset(self.scroll_slider_random_radius_px)
		final_x += offset_x
		final_y += offset_y
	}

	// 7. 计算目标 Y 坐标
	target_y := final_y + (direction * self.scroll_slider_speed_px)

	// 8. 检查边界, 如果撞到边界则重置到中心
	if target_y < self.scroll_slider_bound_up {
		target_y = self.scroll_slider_init_y
	} else if target_y > self.scroll_slider_bound_down {
		target_y = self.scroll_slider_init_y
	}

	// 9. 更新状态
	self.scroll_slider_current_y = target_y
	final_y = target_y

	// 10. 应用曲线
	if self.scroll_slider_curve_enable {
		// (简单曲线, 仅X轴)
		curve_offset := math.Sin(float64(final_y-self.scroll_slider_init_y)*0.1) * float64(self.scroll_slider_curve_amount_px)
		final_x += int32(curve_offset)
	}

	// 11. 应用触摸
	if self.scroll_slider_id == -1 {
		self.scroll_slider_id = self.touch_require(final_x, final_y, touch_pos_scale)
	} else {
		self.touch_move(self.scroll_slider_id, final_x, final_y, touch_pos_scale)
	}
}

// --- [新增 V1.3.0] 结束 ---

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
			// --- [修改 V1.3.0] 使用新的滚轮滑块逻辑 ---
			// (Wheel > 0 (1) 为上, < 0 (-1) 为下)
			// (handel_scroll_slider 需要 -1 为上, 1 为下)
			go self.handel_scroll_slider(Wheel * -1)
			// --- [修改 V1.3.0] 结束 ---
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
			// --- [修改 V1.3.4] 修复: P0 崩溃 ---
			if tid, ok := state.(int32); ok {
				self.touch_release(tid)
			} else {
				logger.Warnf("key[%s] (PRESS) 状态类型错误, 期望 int32, 得到 %T", key_name, state)
			}
			self.key_action_state_save.Delete(key_name)
			// --- [修改 V1.3.4] 修复结束 ---
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
			// --- [修改 V1.3.4] 修复: P0 崩溃 ---
			if release_signal, ok := state.(chan bool); ok {
				release_signal <- true
			} else {
				logger.Warnf("key[%s] (MULT_PRESS) 状态类型错误, 期望 chan bool, 得到 %T", key_name, state)
			}
			// --- [修改 V1.3.4] 修复结束 ---
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
				end_x := int32(action.Get("POS_S").GetIndex(pos_len-1).GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				end_y := int32(action.Get("POS_S").GetIndex(pos_len-1).GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				// --- [修改 V1.1.1] 应用按键抖动 ---
				end_x_jit, end_y_jit := self.apply_key_jitter(end_x, end_y)
				self.touch_move(tid, end_x_jit, end_y_jit, touch_pos_scale)
				// --- [修改 V1.1.1] 结束 ---
				self.touch_release(tid)
			})()
		} else if up_down == UP {

		}

	// --- [新增 V1.3.0] 新按键类型 ---
	case "SYNC_VIEW_RESET": // 同步按抬鼠标重置 (小眼睛)
		// --- [修改 V1.3.4] 修复: P2 视角BUG, 弃用全局 view_is_saved ---
		if up_down == DOWN {
			// 1. 保存当前视角位置 (作为状态)
			saved_pos := [2]int32{self.view_current_x, self.view_current_y}
			self.key_action_state_save.Store(key_name, saved_pos)

			// 2. 计算新位置 (已缩放)
			new_x := int32(action.Get("POS").GetIndex(0).MustFloat64() * float64(self.screen_x))
			new_y := int32(action.Get("POS").GetIndex(1).MustFloat64() * float64(self.screen_y))

			// 3. 重置到新位置 (函数内置了随机)
			self.reset_view_position(new_x, new_y)
		} else if up_down == UP {
			// 1. 恢复到保存的位置
			if saved_state, ok := state.([2]int32); ok {
				self.reset_view_position(saved_state[0], saved_state[1])
			} else {
				// 状态丢失或类型错误, 恢复到 init
				self.reset_view_position(self.view_init_x, self.view_init_y)
				logger.Warnf("key[%s] (SYNC_VIEW_RESET) 状态类型错误, 期望 [2]int32, 得到 %T", key_name, state)
			}
			self.key_action_state_save.Delete(key_name)
		}
		// --- [修改 V1.3.4] 修复结束 ---

	case "CLICK_VIEW_RESET": // 单点击鼠标重置 (药品)
		// --- [修改 V1.3.4] 修复: P2 视角BUG, 弃用全局 view_is_saved ---
		if up_down == DOWN {
			if state == nil { // 第一次按下
				// 1. 保存当前位置 (作为状态)
				saved_pos := [2]int32{self.view_current_x, self.view_current_y}
				self.key_action_state_save.Store(key_name, saved_pos)

				// 2. 计算新位置 (已缩放)
				new_x := int32(action.Get("POS").GetIndex(0).MustFloat64() * float64(self.screen_x))
				new_y := int32(action.Get("POS").GetIndex(1).MustFloat64() * float64(self.screen_y))

				// 3. 重置到新位置
				self.reset_view_position(new_x, new_y)
			} else { // 第二次按下
				// 1. 恢复位置
				if saved_state, ok := state.([2]int32); ok {
					self.reset_view_position(saved_state[0], saved_state[1])
				} else {
					// 状态丢失或类型错误, 恢复到 init
					self.reset_view_position(self.view_init_x, self.view_init_y)
					logger.Warnf("key[%s] (CLICK_VIEW_RESET) 状态类型错误, 期望 [2]int32, 得到 %T", key_name, state)
				}

				// 2. 删除状态
				self.key_action_state_save.Delete(key_name)
			}
		}
		// --- [修改 V1.3.4] 修复结束 ---

	case "BACKPACK_TOGGLE": // 背包键
		if up_down == DOWN {
			// --- [修改 V1.3.4] 修复: P3 逻辑顺序 (先切换, 后点击) ---
			is_open := false
			if state != nil {
				// --- [修改 V1.3.4] 修复: P0 崩溃 (虽然这里是 bool, 但保持一致性) ---
				if v, ok := state.(bool); ok {
					is_open = v
				}
				// --- [修改 V1.3.4] 修复结束 ---
			}

			var pos_to_click *simplejson.Json
			if !is_open {
				// 即将打开背包, 点击 POS (A点)
				pos_to_click = action.Get("POS")
			} else {
				// 即将关闭背包, 点击 POS_B (B点)
				pos_to_click = action.Get("POS_B")
			}

			// 1. 立即切换映射
			self.switch_map_mode()

			// 2. 保存新状态
			self.key_action_state_save.Store(key_name, !is_open)

			// 3. 执行点击 (在 goroutine 中)
			go (func() {
				x := int32(pos_to_click.GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				y := int32(pos_to_click.GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				x_jit, y_jit := self.apply_key_jitter(x, y) // (未缩放)
				tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
				time.Sleep(time.Duration(8) * time.Millisecond)
				self.touch_release(tid)
			})()
			// --- [修改 V1.3.4] 修复结束 ---
		}

	case "CLICK_MAP_ON": // 开启映射后点击（一次）
		if up_down == DOWN {
			// --- [修改 V1.3.4] 修复: P3 逻辑顺序 (先开图, 后点击) ---
			// 1. 立即开启映射 (如果未开)
			if !self.map_on {
				self.switch_map_mode()
			}
			// 2. 执行点击 (复制 CLICK 逻辑)
			// --- [修改 V1.3.4] 修复结束 ---
			go (func() {
				x := int32(action.Get("POS").GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				y := int32(action.Get("POS").GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				x_jit, y_jit := self.apply_key_jitter(x, y)
				tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
				time.Sleep(time.Duration(8) * time.Millisecond)
				self.touch_release(tid)
			})()
		}

	case "CLICK_MAP_OFF": // 点击关闭映射
		if up_down == DOWN {
			// 1. 立即关闭映射 (如果已开)
			if self.map_on {
				self.switch_map_mode()
			}
			// 2. 执行点击 (复制 CLICK 逻辑)
			go (func() {
				x := int32(action.Get("POS").GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				y := int32(action.Get("POS").GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				x_jit, y_jit := self.apply_key_jitter(x, y)
				tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
				time.Sleep(time.Duration(8) * time.Millisecond)
				self.touch_release(tid)
			})()
		}

	case "SEQUENTIAL_PRESS": // 依次触摸点
		if up_down == DOWN {
			// 1. 获取当前索引
			current_index := 0
			if state != nil {
				// --- [修改 V1.3.4] 修复: P0 崩溃 (虽然这里是 int, 但保持一致性) ---
				if v, ok := state.(int); ok {
					current_index = v
				}
				// --- [修改 V1.3.4] 修复结束 ---
			}

			// 2. 构建完整的位置列表 (POS + POS_S)
			all_pos := []*simplejson.Json{action.Get("POS")}
			pos_s_list, ok := action.CheckGet("POS_S")
			if ok {
				pos_s_array := pos_s_list.MustArray()
				for i := range pos_s_array {
					all_pos = append(all_pos, action.Get("POS_S").GetIndex(i))
				}
			}

			total_len := len(all_pos)

			// 3. 获取要点击的位置
			pos_to_click := all_pos[current_index]

			// 4. 计算并保存下一个索引
			next_index := (current_index + 1) % total_len
			self.key_action_state_save.Store(key_name, next_index)

			// 5. 执行点击
			go (func() {
				x := int32(pos_to_click.GetIndex(0).MustFloat64()*float64(self.rel_screen_x))
				y := int32(pos_to_click.GetIndex(1).MustFloat64()*float64(self.rel_screen_y))
				x_jit, y_jit := self.apply_key_jitter(x, y)
				tid := self.touch_require(x_jit, y_jit, touch_pos_scale)
				time.Sleep(time.Duration(8) * time.Millisecond)
				self.touch_release(tid)
			})()
		}
		// --- [新增 V1.3.0] 结束 ---

	}
}

func (self *TouchHandler) switch_map_mode() {
	self.total_move_x = 0
	self.total_move_y = 0 //总移动距离清零
	self.view_id = self.touch_release(self.view_id) //视角id释放

	// --- [修改 V1.3.5] P7: 修复 "物理" 粘滞按键 ---
	self.real_key_down_state.Range(func(key, value interface{}) bool {
		if code, ok := friendly_name_2_keycode[key.(string)]; ok {
			// 强制发送一个 UP (抬起) 事件到 uinput
			self.u_input_control(UInput_key_event, int32(code), UP)
			logger.Debugf("key[%s] 已强制释放 (uinput)", key.(string))
		}
		self.real_key_down_state.Delete(key) // 清理状态
		return true
	})
	// --- [修改 V1.3.5] P7 结束 ---

	self.key_action_state_save.Range(func(key, value interface{}) bool {
		// --- [修改 V1.3.4] 修复: P0 崩溃 (在释放时也检查) ---
		if action, ok := self.config.Get("KEY_MAPS").CheckGet(key.(string)); ok {
			self.execute_key_action(time.Now(), key.(string), UP, action, value)
			logger.Infof("已释放key:%s", key.(string))
		}
		// --- [修改 V1.3.4] 修复结束 ---
		return true
	})

	self.map_on = !self.map_on     //切换
	self.map_switch_signal <- true //发送信号到v_mouse切换显示

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

	// --- [新增 V1.3.4] 修复: P1 拦截问题, 允许特定按键在映射关闭时触发 ---
	if !self.map_on {
		if action, ok := self.config.Get("KEY_MAPS").CheckGet(key_name); ok {
			action_type := action.Get("TYPE").MustString()
			// 只在映射关闭时检查 CLICK_MAP_ON 和 BACKPACK_TOGGLE
			if action_type == "CLICK_MAP_ON" || action_type == "BACKPACK_TOGGLE" {
				if up_down == DOWN { // 只在按下时触发
					state, _ := self.key_action_state_save.Load(key_name)
					self.execute_key_action(time.Now(), key_name, up_down, action, state)
				}
				return // 拦截, 不再继续 (防止进入下面的 u_input_control)
			}
		}
		// (如果不是那两个键, 且映射关闭, 则继续执行下面的 u_input 逻辑)
	}
	// --- [新增 V1.3.4] 修复结束 ---

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
			// --- [修改 V1.2.0] 新 Shift 逻辑 ---
			if up_down == DOWN {
				if self.shift_press_toggle {
					self.wasd_up_down_statues[4] = !self.wasd_up_down_statues[4] // 切换
				} else {
					self.wasd_up_down_statues[4] = true // 长按
				}
			} else if up_down == UP {
				if self.shift_release_toggle {
					self.wasd_up_down_statues[4] = !self.wasd_up_down_statues[4] // 切换
				} else if !self.shift_press_toggle { // 仅在非"按下切换"模式时，抬起才恢复
					self.wasd_up_down_statues[4] = false // 长按释放
				}
			}
			// --- [修改 V1.2.0] 结束 ---
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
				// --- [修改 V1.3.4] 修复: P4 背包键BUG, 允许重复按下 ---
				action_type := action.Get("TYPE").MustString()
				if action_type != "CLICK_VIEW_RESET" &&
					action_type != "SEQUENTIAL_PRESS" &&
					action_type != "BACKPACK_TOGGLE" && // (P4)
					action_type != "CLICK_MAP_ON" { // (P4)
					// 原版逻辑: 阻止 PRESS/AUTO_FIRE 等重复触发
				} else {
					// [V1.3.4] 新逻辑: 允许重复 DOWN
					self.execute_key_action(time.Now(), key_name, up_down, action, state)
				}
				// --- [修改 V1.3.4] 修复结束 ---
			} else {
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

				// --- [新增 V1.3.5] P7: 跟踪物理按键状态 ---
				if up_down == DOWN {
					self.real_key_down_state.Store(key_name, true)
				} else {
					self.real_key_down_state.Delete(key_name)
				}
				// --- [新增 V1.3.5] 结束 ---
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
				self.abs_last.Store(name, formatted_value)
				//右摇杆控制视角 只需修改值 有单独线程去处理

				// [V1.2.5 回滚] 回滚到 V1.2.3 的摇杆逻辑 (解决 V1.2.5 的“恒星一直转”BUG)
				if (name == "LS_X" || name == "LS_Y") && self.map_on {
					ls_x, ls_y := self.getStick("LS")
					if ls_x == 0.5 && ls_y == 0.5 {
						// [V1.2.3] 摇杆停止移动
						if self.ls_wheel_released == false {
							self.ls_wheel_released = true
							self.star_curve_counter = 0 // [V1.2.3] 重置恒星曲线计数器
						}
					} else {
						// [V1.2.3] 摇杆正在移动
						self.ls_wheel_released = false

						// [V1.2.5 回滚] 使用 V1.2.3 的摇杆“方形”逻辑
						wheel_range := self.wheel_range
						if self.wasd_up_down_statues[4] {
							wheel_range = self.wheel_shift_range
						}
						// 1. [V1.2.3] 计算“主路径”目标点
						target_x := self.wheel_init_x + int32(float64(wheel_range)*2*(ls_x-0.5)) //注意这里的X和Y是相反的
						target_y := self.wheel_init_y + int32(float64(wheel_range)*2*(ls_y-0.5))

						// 2. [V1.2.3] 计算“恒星曲线”的偏移量
						curve_x, curve_y := self.get_star_curve_offset()

						// 3. [V1.2.3] 施加曲线
						final_x := target_x + curve_x
						final_y := target_y + curve_y

						// 4. [V1.2.3] 更新恒星位置 (不使用平滑)
						self.handel_wheel_action(Wheel_action_move, final_x, final_y)
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
			return y, wm_size_x - x
		case 2: //up side down
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

			logger.Debugf("rel_event\t%v \n", rel_sin)
			logger.Debugf("key_events\t%v \n", key_sin)
			logger.Debugf("abs_events\t%v \n", abs_sin)

		}
	}
}
