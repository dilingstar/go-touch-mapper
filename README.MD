加qq群交流:[1067729485](http://qm.qq.com/cgi-bin/qm/qr?_wv=1027&k=YvlipEC3TLgLDKu843aRyzKaiHi_Naw0&authKey=f2RLVU4bdbYsuvSSNN1FBbGbFcAuxG30iY2UIDhMGNM7Pa4fsIlf0YNvrg7n5%2F%2Bf&noverify=0&group_code=1067729485)


7# 安卓触屏映射工具

读取手柄，键盘，鼠标，触摸屏等输入并转换为虚拟触屏设备输出

[视频教程](https://www.bilibili.com/video/BV1XtgeztEvY/)

## Quick start

从release直接下载编译好的二进制程序

一般手机平板下载 [go-touch-mapper](https://github.com/dilingstar/go-touch-mapper/releases/download/V1.1.0/go-touch-mapper)

androidX86下载 [go-touch-mapper]
或者使用国内镜像

一般手机平板 [go-touch-mapper]

androidX86 [go-touch-mapper]

读取dev和创建uinput需要shell或者root权限

使用adb shell或者su shell运行

```
cd /data/local/tmp && \ 
curl -LJo go-touch-mapper https://vip.123pan.cn/1833788059/direct/projects/go-touch-mapper/build/go-touch-mapper_arm64 && \
chmod 777 ./go-touch-mapper && \
./go-touch-mapper -h
```
如果提示无curl，则需要手动下载后push到/data/local/tmp下运行

## Build & Run

可在安卓设备的termux环境下自己编译
```
pkg i golang yarn -y
git clone https://github.com/RiderLty/go-touch-mapper
cd go-touch-mapper
cd go-touch-mapper-gh-pages
yarn install
yarn build
cd ..
CGO_ENABLE=0 GOOS=linux go build -ldflags="-s -w"
./go-touch-mapper -h
```

## Help

```
usage: go-touch-mapper [-h|--help] [-c|--config "<value>"] [--create-js-info]
                       [--pattern "<value>"] [-t|--touch-disabled]
                       [-i|--inputManager <integer>] [--tty-path "<value>"]
                       [--tty-screen "<value>"] [-r|--remoteControl] [-p|--port
                       <integer>] [-v|--v-mouse] [--auto-release <integer>]
                       [--measure-mode] [-d|--debug_mode]



Arguments:

  -h  --help            Print help information
  -c  --config
                        指定配置文件路径，如果不存在则会使用默认模板创建
      --create-js-info  创建手柄配置文件模式. Default: false
      --pattern         用于筛选设备名称的正则. Default: .*
  -t  --touch-disabled  关闭触屏混合. Default: false
  -i  --inputManager    DisplayID
                        使用inputManager控制触摸，可用于控制外接显示器
                        . Default: -1
      --tty-path        串口设备路径
                        将使用串口控制外接的HID设备模拟触屏.
                        Default:
      --tty-screen
                        使用串口控制HID设备模拟触屏的屏幕参数，宽x高x旋转方向，例如1440x3440x1.
                        Default: 1440x3440x1
  -r  --remoteControl   是否从UDP接收远程事件. Default: false
  -p  --port
                        指定监听远程事件的UDP端口号与控制后台端口.
                        Default: 61069
  -v  --v-mouse
                        用触摸操作模拟鼠标,需要额光标外显示程序.
                        Default: false
      --auto-release
                        触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用.
                        Default: 200
      --measure-mode
                        显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调.
                        Default: false
  -d  --debug_mode      打印debug信息. Default: false
```
## 运行示例

```
./go-touch-mapper -c df.json
```

如果df.json不存在，会从模板创建配置文件。


## 映射开关
默认键盘使用 ` 键，可在配置文件中手动修改，手柄固定为SELECT+RS组合键

## 配置文件

程序运行后会显示控制后台地址```http://[ip]:61070/```用于编辑配置文件。

如有需要，可参照[EXAMPLE.JSON](https://github.com/dilingstar/go-touch-mapper/blob/main/configs/EXAMPLE.JSON)手动修改配置文件

可自动检测屏幕旋转方向，始终以左上角为原点，向右为X，向下为Y

按住键盘或者手柄按键后，点击屏幕对应位置即可添加按键映射

灵敏度设置 MOUSE:SPEED:[ x , y ] float64类型，此数值也会影响右摇杆控制视角的速度

实际计算输出为 int( Speed * 256 * real_move ) ，所以建议最小值大于0.00390625，否则可能会出现精度丢失的情况

虚拟触屏的大小为原始屏幕尺寸的256倍，用于解决缩放时慢速移动累计精度丢失的问题

## 触屏混合

程序会创建额外的uinput虚拟触屏来进行输入，默认会识别并读取机器物理触屏输入，与键鼠混合后使用虚拟触屏输出以实现映射开启时，触摸屏幕不中断操作。

可使用-t禁用此功能

⚠ 如果是在手机端进行配置文件的修改，建议禁用，避免无法操作手机。

## 手柄支持
joystickInfos文件夹下json文件为手柄配置文件

按照按键位置统一为XBOX手柄按键布局，可直接下载使用。

扳机键按照力度映射为5个等级的按键 BTN_LT_1 ... BTN_LT_5，等级5同时会出发BTN_LT控制信号。

没有线性扳机的手柄例如 Switch Pro controller 只有 BTN_LT 和 BTN_RT

  注意⚠ : 

* 手柄的按键代码在不同设备上可能有所差异
* 如果出现按键无反应的情况，建议重新创建手柄配置文件

### 手动创建手柄配置文件
```
./go-touch-mapper --create-js-info 
```

## 远程控制

添加 -r 参数

```
./go-touch-mapper -r
```

PC上 修改remote_control中sender("192.168.1.64:61069")为手机IP，端口默认可不修改

或者下载[打包的exe](remote_control/dist/remote_control.exe)

在同目录创建[addr.txt](remote_control/dist/addr.txt)，写入要控制设备的IP:端口

运行

``` 
python remote_control.py 
```

## HID触屏

使用ESP32-s3的HID库，创建一个外置触屏，在上位机通过串口通信的方式发送触摸指令

优点在于无需在被控设备上安装任何软件或者执行指令

### 使用方式

* 刷入hid_touch.ino固件   [在线烧录](https://riderlty.github.io/go-touch-mapper/)

* 将键鼠手柄与ESP32-s3的串口连接到例如树莓派之类的linux设备上，USB口连接到被控设备

* 执行 ```ls /dev | grep ttyACM``` 查看串口设备

* 使用--tty-path指定串口设备，--tty-screen 执行屏幕尺寸与旋转方向，例如：```./go-touch-mapper--tty-path /dev/ttyACM0 --tty-screen 1920x1080x1``` 

* 建议设备回报率不要超过500HZ

## [InputManager](https://github.com/RiderLty/inputManager-touch-interface)
### 说明
由Scrcpy修改而来，使用app_process启动

通过 Unix Domain Socket 与主程序通信控制触屏

在多显示器情况下使用```-i```指定displayID

只可在adb shell下使用

## ADB 下载

[platform-tools-latest-windows](https://vip.123pan.cn/1833788059/23477218)
