import HighlightOffIcon from '@mui/icons-material/HighlightOff';
import { Button, FormControlLabel, IconButton, Input, Paper, Slider, Switch, Typography } from "@mui/material";
import FormControl from '@mui/material/FormControl';
import Grid from '@mui/material/Grid';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
// import ControlPanel from "./ControlPanel";
import DraggableContainer from "./DraggableContainer";
import JoystickListener from "./JoystickListener";
import * as keyNameMap from "./keynamemap.json";


import {
    UploadButton,
    UploadButtonJIETU,
    UploadButton5s,
    FixedIcon,
    GroupFixedIcon,
    CostumedInput,
    WheelShow,
    ViewShow,
    // --- [新增 V1.3.1] 导入新组件 ---
    ViewResetRadiusShow,
    ScrollSliderShow,
    // --- [新增 V1.3.1] 结束 ---
} from "./UIcomponents"
import { produce } from "immer"
import FullscreenIcon from '@mui/icons-material/Fullscreen';

function copyToClipboard(text) {
    let transfer = document.createElement('input');
    document.body.appendChild(transfer);
    transfer.value = text;  // 这里表示想要复制的内容
    transfer.focus();
    transfer.select();
    if (document.execCommand('copy')) {
        document.execCommand('copy');
    }
    transfer.blur();
    document.body.removeChild(transfer);
}


function imageUrlToBase64(url) {
    return new Promise((resolve, reject) => {
        // 1. 创建 Image 对象
        const img = new Image();

        // 2. 设置跨域处理（如果需要）
        img.crossOrigin = "Anonymous";

        // 3. 加载图片
        img.src = url;

        img.onload = () => {
            // 4. 创建 Canvas 元素
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');

            // 5. 设置 Canvas 尺寸与图片相同
            canvas.width = img.width;
            canvas.height = img.height;

            // 6. 将图片绘制到 Canvas 上
            ctx.drawImage(img, 0, 0);

            try {
                // 7. 转换为 Base64 字符串
                const base64String = canvas.toDataURL('image/png');
                resolve(base64String);
            } catch (e) {
                reject(`转换失败: ${e}`);
            }
        };

        img.onerror = (err) => {
            reject(`图片加载失败: ${err}`);
        };
    });
}


async function getImageObjectUrl(src) {
    try {
        // 1. 使用fetch获取图片资源
        const response = await fetch(src);

        // 2. 检查响应状态
        if (!response.ok) {
            throw new Error(`图片加载失败，状态码: ${response.status}`);
        }

        // 3. 获取图片Blob数据
        const blob = await response.blob();

        // 4. 验证是否为图片类型
        if (!blob.type.startsWith('image/')) {
            throw new Error('获取的资源不是图片类型');
        }

        // 5. 生成并返回Object URL
        return URL.createObjectURL(blob);
    } catch (error) {
        // 6. 错误处理
        console.error('获取图片URL失败:', error);
        throw error; // 可选择重新抛出错误或返回fallback URL
    }
}


// --- [新增 V1.0.1] 组合组件 (需求 #5) ---
// 为了满足 "滑块+输入框" 的需求，我们创建一个可重用的组件
// 它被定义在 ConfigManager 外部，以避免不必要的重渲染
const SliderWithInput = ({ label, value, onChange, min, max, step, disabled = false, width = "50px" }) => {

    const handleSliderChange = (event, newValue) => {
        onChange(newValue);
    };

    const handleInputChange = (newValue) => {
        // --- [修改 V1.1.0] 使用 parseFloat 支持小数 ---
        let numValue = parseFloat(newValue);
        if (isNaN(numValue)) return;
        if (numValue < min) numValue = min;
        if (numValue > max) numValue = max;
        onChange(numValue);
    };

    // 使用 Grid 来布局： [标签] [---滑动条---] [输入框]
    return (
        <Grid container direction="row" justifyContent="space-between" alignItems="center" spacing={1} sx={{ mt: 0, mb: 0, height: "40px" }}>
            <Grid item xs={3} sx={{ pr: 0 }}>
                <Typography gutterBottom sx={{ minWidth: "70px", fontSize: "0.9rem", whiteSpace: "nowrap" }}>
                    {label}
                </Typography>
            </Grid>
            <Grid item xs={6} sx={{ pl: 0, pr: 0 }}>
                <Slider
                    disabled={disabled}
                    min={min}
                    max={max}
                    step={step}
                    value={value}
                    valueLabelDisplay="auto"
                    onChange={handleSliderChange}
                    size="small"
                />
            </Grid>
            <Grid item xs={3} sx={{ pl: 1 }}>
                <CostumedInput
                    key={value + (disabled ? '-disabled' : '')} // 关键: 当滑块改变值或禁用状态改变时，强制重渲染输入框
                    defaultValue={value}
                    onCommit={handleInputChange}
                    width={width}
                    disabled={disabled}
                    // --- [修改 V1.1.0] 允许输入小数点 ---
                    type="number" // Use type="number" for native validation if needed
                    inputProps={{ step: step }} // Allow decimal steps
                />
            </Grid>
        </Grid>
    );
};
// --- [新增 V1.0.1] 组合组件结束 ---


export default function ConfigManager() {
    // 屏幕会自适应旋转方向，始终以观看者左上角为原点，向右为x，向下为y
    //所有坐标均为浮点数，真实值为数值*对应方向的屏幕尺寸
    //单向的量，比如轮盘半径，以宽度为标量

    // --- [修改 V1.3.5] 初始化 state, 增加 P2 和 P5 的新字段 ---
    const [config, setConfig] = useState({
        "SCREEN": {
            "SIZE": [
                3200,
                1440
            ]
        },
        "MOUSE": {
            "SWITCH_KEYS": ["KEY_GRAVE"],
            "POS": [
                0.52,
                0.5
            ],
            "SPEED": [
                0.3,
                0.3
            ],
            "VIEW_AUTO_RELEASE_ENABLE": false, // [V1.3.5] P5 新增
            "VIEW_AUTO_RELEASE_MS": 200,     // [V1.3.5] P5 修改
            "VIEW_RESET_RADIUS_ENABLE": false,
            "VIEW_RESET_RADIUS": 0.1,
            "VIEW_RESET_RADIUS_THICKNESS": 0.005, // [V1.3.5] P2 新增
            "VIEW_RANDOM_RESET_ENABLE": false,
            "VIEW_RANDOM_RESET_RADIUS": 0.01
        },
        "WHEEL": {
            "POS": [
                0.17395833333333333,
                0.7361111111111112
            ],
            "RANGE": 0.05,
            "SHIFT_RANGE": 0.11,
            "SHIFT_RANGE_ENABLE": true,
            "SHIFT_PRESS_TOGGLE": false,
            "SHIFT_RELEASE_TOGGLE": false,
            "WASD": [
                "KEY_W",
                "KEY_A",
                "KEY_S",
                "KEY_D"
            ],
            "STEP_SPEED": 60,
            "STAR_DYNAMIC_SPEED": { // V1.2.3 重命名
                "ENABLE": false,
                "MIN_SPEED": 10.0,
                "FREQUENCY": 1.0
            },
            "RANDOM_START": {
                "ENABLE": false,
                "RADIUS": 0.01
            },
            "WHEEL_PLANET": {
                "ENABLE": false,
                "RADIUS": 0.015,
                "SPEED": 1.5,
                "PLANET_DYNAMIC_SPEED": {
                    "ENABLE": false,
                    "MIN_SPEED": 0.5,
                    "FREQUENCY": 1.0
                }
            },
            "PLANET_CURVE": { // V1.2.3 重命名
                "ENABLE": false,
                "CURVE_AMOUNT": 0.005,
                "CURVE_FREQUENCY": 1.0
            },
            "STAR_CURVE": { // V1.2.3 重命名
                "ENABLE": false,
                "CURVE_AMOUNT": 0.002,
                "CURVE_FREQUENCY": 1.0
            }
        },
        "SCROLL_SLIDER": {
            "ENABLE": false,
            "POS": [
                0.9,
                0.5
            ],
            "LENGTH_UP": 0.2,
            "LENGTH_DOWN": 0.2,
            "TIMEOUT_S": 3,
            "SPEED": 1.0,
            "RANDOM_START_ENABLE": false,
            "RANDOM_START_RADIUS": 0.005,
            "CURVE_ENABLE": false,
            "CURVE_AMOUNT": 0.005
        },
        "KEY_JITTER": {
            "ENABLE": true,
            "AMOUNT": 0.003
        },
        "KEY_MAPS": {
            "BTN_LEFT": {
                "TYPE": "PRESS",
                "POS": [
                    0.08333333333333333,
                    0.49074074074074076
                ]
            },
            "BTN_RIGHT": {
                "TYPE": "PRESS",
                "POS": [
                    0.9307291666666667,
                    0.5370370370370371
                ]
            }
        },
        "IMG": "data:image/webp;base64,UklGRoIiAABXRUJQVlA4WAoAAAAoAAAAfwwAnwUASUNDUMgBAAAAAAHIAAAAAAQwAABtbnRyUkdCIFhZWiAH4AABAAEAAAAAAABhY3NwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAA9tYAAQAAAADTLQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAlkZXNjAAAA8AAAACRyWFlaAAABFAAAABRnWFlaAAABKAAAABRiWFlaAAABPAAAABR3dHB0AAABUAAAABRyVFJDAAABZAAAAChnVFJDAAABZAAAAChiVFJDAAABZAAAAChjcHJ0AAABjAAAADxtbHVjAAAAAAAAAAEAAAAMZW5VUwAAAAgAAAAcAHMAUgBHAEJYWVogAAAAAAAAb6IAADj1AAADkFhZWiAAAAAAAABimQAAt4UAABjaWFlaIAAAAAAAACSgAAAPhAAAts9YWVogAAAAAAAA9tYAAQAAAADTLXBhcmEAAAAAAAQAAAACZmYAAPKnAAANWQAAE9AAAApbAAAAAAAAAABtbHVjAAAAAAAAAAEAAAAMZW5VUwAAACAAAAAcAEcAbwBvAGcAbABlACAASQBuAGMALgAgADIAMAAxADZWUDggRiAAAFDMA50BKoAMoAU+MRiMRKIhoRAEACADBLS3cLuwj24D8AAACs3a8XJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtk5D32ych77ZOQ99snIe+2TkPfbJyHvtWAAP7/YcP//79pe+0vfaX+vb///TZv02b9Nm/9MWAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEVYSUZGAAAATU0AKgAAAAgABAEAAAQAAAABAAAMgAEBAAQAAAABAAAFoAESAAMAAAABAAEAAIdpAAQAAAABAAAAPgAAAAAAAAAAAAAAAA=="
    })
    // --- [修改 V1.3.5] 初始化 state 完毕 ---


    const [exportButtonText, setExportButtonText] = useState("更新配置")
    const [selectKEY, setSelectKEY] = useState(null)

    // const [uploadButton, setUploadButton] = useState(true);
    const [imgUrl, setImgUrl] = useState(config["IMG"]);

    const [imgSize, setImgSize] = useState([1, 1])
    const getDisplayValueX = useCallback((value) => { return parseInt(value * config["SCREEN"]["SIZE"][0]) }, [config["SCREEN"]["SIZE"]])
    const getDisplayValueY = useCallback((value) => { return parseInt(value * config["SCREEN"]["SIZE"][1]) }, [config["SCREEN"]["SIZE"]])
    const getPostionValueX = useCallback((value) => { return parseInt(value * imgSize[0]) }, [imgSize])
    const getPostionValueY = useCallback((value) => { return parseInt(value * imgSize[1]) }, [imgSize])

    const viewCenterSetting = useRef(false)
    // [V1.3.0] 新增滚轮滑块位置设置
    const scrollSliderPosSelecting = useRef(false)
    const addingSwitchKey = useRef(false)
    const [addingSwitchKeyInfoText, setAddingSwitchKeyInfoText] = useState("添加映射切换键")

    const handleFileChange = (e) => {
        // setUploadButton(false);
        const reads = new FileReader();
        reads.readAsDataURL(document.getElementById('fileInput').files[0]);
        reads.onload = function (e) {
            setImgUrl(this.result);
            document.body.requestFullscreen();
        };
    }

    const getRemoteApiImg = async (url) => {
        // const objurl = await getImageObjectUrl(url)
        // setImgUrl(objurl)
        // document.body.requestFullscreen();
        const bas64STR = await imageUrlToBase64(url)
        setConfig(produce(draft => { draft.IMG = bas64STR }))
        document.body.requestFullscreen();
    }

    const imgLoaded = () => {
        setImgSize([document.getElementById("img").width, document.getElementById("img").height])
        setConfig(produce(draft => { draft.SCREEN.SIZE = [document.getElementById("img").naturalWidth, document.getElementById("img").naturalHeight] }))
    }



    const handelImgClick = (e) => {
        const rect = document.getElementById("img").getBoundingClientRect()
        const key = selectKEY
        const x = (e.clientX - rect.left) / document.getElementById("img").width;
        const y = (e.clientY - rect.top) / document.getElementById("img").height
        if (x > 1 || y > 1) {//忽略大于屏幕的
            return
        }
        if (viewCenterSetting.current) {
            setConfig(produce(draft => {
                draft.MOUSE.POS = [
                    x, y
                ]
            }))
            viewCenterSetting.current = false
            return
        }
        
        // --- [V1.3.0] 新增：滚轮滑块位置设置 ---
        if (scrollSliderPosSelecting.current) {
            setConfig(produce(draft => {
                draft.SCROLL_SLIDER.POS = [ x, y ]
            }))
            scrollSliderPosSelecting.current = false
            return
        }
        // --- [V1.3.0] 结束 ---


        if (key !== null) {
            if (key === "REL_WHEEL_UP" || key == "REL_WHEEL_DOWN") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[key] = {
                        "TYPE": "CLICK",
                        "POS": [
                            x,
                            y
                        ],
                        "INTERVAL": [18]
                    }
                }))
            } else {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[key] = {
                        "TYPE": "PRESS",
                        "POS": [
                            x,
                            y
                        ]
                    }
                }))
            }
            if (["BTN_LEFT", "BTN_MIDDLE", "BTN_RIGHT", "BTN_SIDE", "BTN_EXTRA", "REL_WHEEL_DOWN", "REL_WHEEL_UP"].indexOf(key) !== -1) {
                setSelectKEY(null)
            }
        } else {
            if (window.dispatchEvent) {
                window.dispatchEvent(new CustomEvent('imgOnNoKeyClick', {
                    detail: { x: x, y: y }
                }))
            } else {
                window.fireEvent(new CustomEvent('imgOnNoKeyClick', {
                    detail: { x: x, y: y }
                }));
            }

        }
    }

    const exportJSON = () => {
        // copyToClipboard(JSON.stringify(config))
        setExportButtonText("配置更新中")
        fetch('/configure/set', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        }).then(resp => resp.text()).then(text => {
            setExportButtonText(text)
            setTimeout(() => {
                setExportButtonText("更新配置")
            }, 1000)
        }).catch(err => {
            setExportButtonText(String(err))
            setTimeout(() => {
                setExportButtonText("更新配置")
            }, 1000)
        })
    }
    const OtherSettings = () => {

        const wheelPosSelecting = useRef(false)
        const [range, setRange] = useState(config["WHEEL"]["RANGE"] * 100)
        const [shiftRange, setShiftRange] = useState(config["WHEEL"]["SHIFT_RANGE"] * 100)

        // [V1.3.0] 滚轮滑块按钮状态
        const [scrollSliderSetPosButtonDisabled, setScrollSliderSetPosButtonDisabled] = useState(false)

        const [setPosButtonDisabled, setSetPosButtonDisabled] = useState(false)
        const readyToSetPos = () => {
            wheelPosSelecting.current = true;
            setSetPosButtonDisabled(true)
        }

        // [V1.3.0] 滚轮滑块设置位置
        const readyToSetScrollSliderPos = () => {
            scrollSliderPosSelecting.current = true;
            setScrollSliderSetPosButtonDisabled(true);
        }

        const imgClickListener = (e) => {
            if (wheelPosSelecting.current) {
                setConfig(produce(draft => { draft.WHEEL.POS = [e.detail.x, e.detail.y] }))
                wheelPosSelecting.current = false
                setSetPosButtonDisabled(false)
            }
            // [V1.3.0] 滚轮滑块
            if (scrollSliderPosSelecting.current) {
                setConfig(produce(draft => { draft.SCROLL_SLIDER.POS = [e.detail.x, e.detail.y] }))
                scrollSliderPosSelecting.current = false
                setScrollSliderSetPosButtonDisabled(false)
            }
        }

        useEffect(() => {

            window.addEventListener('imgOnNoKeyClick', imgClickListener)
            return () => {
                window.removeEventListener('imgOnNoKeyClick', imgClickListener)
            }
        }, [])

        // --- [修改 V1.2.5] 曲线设置的复用组件 (增加 freqMax 属性) ---
        // --- [修改 V1.3.3] 修复 CurveSettings, 严格遵循 V1.3.0 的扁平 SCROLL_SLIDER 结构 ---
        const CurveSettings = ({ curveType, freqMax = 30, configPath }) => { 
            const curveNameMap = {
                "STAR_CURVE": "恒星曲线",
                "PLANET_CURVE": "行星曲线",
                "CURVE": "路径曲线" // [V1.3.0] 滚轮滑块的曲线
            }
            const curveLabel = curveNameMap[curveType] || "曲线";
            
            // [V1.3.3] 判定是否是滚轮滑块
            const isScrollSliderCurve = (configPath === "SCROLL_SLIDER" && curveType === "CURVE");
            
            // [V1.3.3] 动态获取数据
            let curveData;
            let curveEnable;
            let curveAmount;
            let curveFrequency;

            if (isScrollSliderCurve) {
                // 滚轮滑块 (扁平)
                curveData = config.SCROLL_SLIDER;
                curveEnable = curveData.CURVE_ENABLE;
                curveAmount = curveData.CURVE_AMOUNT;
                // V1.3.0 JSON 中没有 frequency, 设为 0
                curveFrequency = 0.0; 
            } else {
                // 轮盘 (嵌套)
                curveData = config.WHEEL[curveType];
                curveEnable = curveData.ENABLE;
                curveAmount = curveData.CURVE_AMOUNT;
                curveFrequency = curveData.CURVE_FREQUENCY;
            }

            // [V1.3.3] 动态设置
            const setCurveConfig = (key, value) => {
                setConfig(produce(draft => {
                    if (isScrollSliderCurve) {
                        // 滚轮滑块 (扁平)
                        draft.SCROLL_SLIDER[key] = value;
                    } else {
                        // 轮盘 (嵌套)
                        draft.WHEEL[curveType][key] = value;
                    }
                }))
            }

            return (
                <>
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 2 }}>
                        <Typography gutterBottom sx={{ minWidth: "100px" }}>
                            {curveLabel}
                        </Typography>
                        <Switch
                            checked={curveEnable}
                            onChange={() => setCurveConfig(isScrollSliderCurve ? "CURVE_ENABLE" : "ENABLE", !curveEnable)}
                        />
                    </Grid>
                    {/* 曲线幅度 (通用) */}
                    <SliderWithInput
                        label={`${curveLabel}幅度`}
                        disabled={!curveEnable}
                        value={curveAmount * 100} // %
                        min={0} max={5} step={0.1}
                        onChange={(value) => setCurveConfig(isScrollSliderCurve ? "CURVE_AMOUNT" : "CURVE_AMOUNT", Number(value) / 100)}
                    />
                    {/* 曲线频率 (仅轮盘) */}
                    {!isScrollSliderCurve && (
                        <SliderWithInput
                            label={`${curveLabel}频率`}
                            disabled={!curveEnable}
                            value={curveFrequency} // Hz
                            min={0.1} 
                            max={freqMax} // [V1.2.5] 使用 freqMax 属性
                            step={0.1}
                            onChange={(value) => setCurveConfig("CURVE_FREQUENCY", Number(value))}
                        />
                    )}
                </>
            );
        };
        // --- [修改 V1.3.3] 曲线设置组件结束 ---


        return <Paper sx={{
            width: "370px",
            marginLeft: "10px",
            paddingBottom: "10px", // 增加一点底部 padding
        }}>
            <Grid
                container
                direction="row"
                justifyContent="space-between"
                alignItems="center"
            >
                <Grid item>
                    {selectKEY ? <a>&emsp;点击屏幕映射{selectKEY}</a> : <a>&emsp;按下某个按键并点击</a>}
                </Grid>
                <Grid item>
                    <IconButton onClick={() => document.body.requestFullscreen()} ><FullscreenIcon /></IconButton>
                </Grid>
            </Grid>

            <Grid
                container
                spacing={"10px"}
                direction="column"
                justify="center"
                alignItems="center"
                sx={{
                    width: "350px",
                    marginLeft: "10px",
                    marginTop: "1px",
                }}
            >
                <Grid
                    container
                    direction="row"
                    justifyContent="space-evenly"
                    alignItems="center"
                    spacing={"10px"}
                >
                    <Grid item xs={6}>
                        <Button
                            onClick={() => { getRemoteApiImg("/screen.png") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                                marginTop: "10px",
                            }}
                        >{"获取截图"}</Button>
                    </Grid>
                    <Grid item xs={6}>
                        <Button
                            onClick={() => { setTimeout(() => { getRemoteApiImg("/screen.png") }, 5000) }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                                marginTop: "10px",
                            }}
                        >{"5s后获取截图"}</Button>

                    </Grid>
                </Grid>
                <Button
                    onClick={exportJSON}
                    variant="outlined"
                    sx={{
                        width: "100%",
                        marginTop: "10px",
                    }}
                >{exportButtonText}</Button>

                <Grid
                    container
                    direction="row"
                    justifyContent="space-evenly"
                    alignItems="center"
                    spacing={"10px"}
                >
                    <Grid item xs={12}>

                        <Typography
                            sx={{
                                width: "100%",
                                marginTop: "10px",
                            }}
                        >
                            鼠标按键
                        </Typography>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            onClick={() => { setSelectKEY("BTN_LEFT") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"左键"}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            onClick={() => { setSelectKEY("BTN_MIDDLE") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"中键"}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            onClick={() => { setSelectKEY("BTN_RIGHT") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"右键"}</Button>
                    </Grid>
                    <Grid item xs={3}>
                        <Button
                            onClick={() => { setSelectKEY("BTN_EXTRA") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"前进"}</Button>
                    </Grid>
                    <Grid item xs={3}>
                        <Button
                            onClick={() => { setSelectKEY("BTN_SIDE") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"后退"}</Button>
                    </Grid>
                    <Grid item xs={3}>
                        <Button
                            onClick={() => { setSelectKEY("REL_WHEEL_UP") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"滚轮上"}</Button>
                    </Grid>
                    <Grid item xs={3}>
                        <Button
                            onClick={() => { setSelectKEY("REL_WHEEL_DOWN") }}
                            variant="outlined"
                            sx={{
                                width: "100%",
                            }}
                        >{"滚轮下"}</Button>
                    </Grid>
                </Grid>

                {/* --- [新 V1.1.1] 按键抖动 (随机落点) --- */}
                <Paper sx={{ width: "100%", p: 1, marginTop: "10px" }}>
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center">
                        <Typography gutterBottom sx={{ minWidth: "160px" }}>
                            按键随机落点
                        </Typography>
                        <Switch
                            checked={config.KEY_JITTER.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.KEY_JITTER.ENABLE = !draft.KEY_JITTER.ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="抖动幅度 (%)"
                        disabled={!config.KEY_JITTER.ENABLE}
                        value={config.KEY_JITTER.AMOUNT * 100} // Convert ratio to %
                        min={0} max={5} step={0.05} // 0% to 5%
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.KEY_JITTER.AMOUNT = Number(value) / 100; })) // Convert % back to ratio
                        }}
                    />
                    <Typography variant="caption" sx={{ marginLeft: "10px" }}>(0.3% 约 10 像素 @ 3200px 宽)</Typography>
                </Paper>
                {/* --- [新 V1.1.1] 结束 --- */}


                {/* [修改 V1.0.0] 分辨率设置 */}
                <Grid
                    container
                    direction="row"
                    justifyContent="flex-start"
                    alignItems="center"
                    sx={{ height: "50px", marginTop: "10px" }}
                >
                    <a>屏幕分辨率&emsp;&emsp;横向 : </a>
                    <CostumedInput
                        key={config["SCREEN"]["SIZE"][0]} // 强制重渲染
                        defaultValue={config["SCREEN"]["SIZE"][0]}
                        onCommit={(value) => {
                            setConfig(produce(draft => { draft.SCREEN.SIZE[0] = Number(value); }))
                        }}
                        width="50px"
                    />
                    <a> &emsp;纵向 : </a>
                    <CostumedInput
                        key={config["SCREEN"]["SIZE"][1]} // 强制重渲染
                        defaultValue={config["SCREEN"]["SIZE"][1]}
                        onCommit={(value) => {
                            setConfig(produce(draft => { draft.SCREEN.SIZE[1] = Number(value); }))
                        }}
                        width="50px"
                    />
                </Grid>


                {/* [修改 V1.3.5] 视角设置 (添加 P2 和 P5 UI) */}
                <Paper sx={{ width: "100%", p: 1, marginTop: "10px" }}>
                    <Typography variant="subtitle1" gutterBottom>
                        视角设置
                    </Typography>
                    <Grid
                        container
                        direction="row"
                        justifyContent="flex-start"
                        alignItems="center"
                        sx={{ height: "50px" }}
                    >
                        <a>视角灵敏度&nbsp;X:</a>
                        <CostumedInput
                            key={config["MOUSE"]["SPEED"][0]} // 强制重渲染
                            defaultValue={config["MOUSE"]["SPEED"][0]}
                            onCommit={(value) => {
                                setConfig(produce(draft => { draft.MOUSE.SPEED[0] = Number(value); }))
                            }} width="40px" />
                        <a>&nbsp;Y:</a>
                        <CostumedInput
                            key={config["MOUSE"]["SPEED"][1]} // 强制重渲染
                            defaultValue={config["MOUSE"]["SPEED"][1]}
                            onCommit={(value) => {
                                setConfig(produce(draft => { draft.MOUSE.SPEED[1] = Number(value); }))
                            }} width="40px" />
                        <a>&emsp;中心:</a>
                        <Button onClick={() => { viewCenterSetting.current = true }} disabled={setPosButtonDisabled || scrollSliderPosSelecting.current} sx={{ height: "30px", marginLeft: "5px" }} variant="outlined"  >重设</Button>
                    </Grid>

                    {/* --- [修改 V1.3.5] P5: 视角自动释放 (开关 + 输入框) --- */}
                    <Grid
                        container
                        direction="row"
                        justifyContent="flex-start"
                        alignItems="center"
                        sx={{ height: "50px" }}
                    >
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            视角自动释放
                        </Typography>
                        <Switch
                            checked={config.MOUSE.VIEW_AUTO_RELEASE_ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.MOUSE.VIEW_AUTO_RELEASE_ENABLE = !draft.MOUSE.VIEW_AUTO_RELEASE_ENABLE; }))
                            }}
                        />
                        <CostumedInput
                            key={config.MOUSE.VIEW_AUTO_RELEASE_MS + (config.MOUSE.VIEW_AUTO_RELEASE_ENABLE ? '' : '-disabled')} // 强制重渲染
                            defaultValue={config.MOUSE.VIEW_AUTO_RELEASE_MS}
                            disabled={!config.MOUSE.VIEW_AUTO_RELEASE_ENABLE}
                            onCommit={(value) => {
                                setConfig(produce(draft => { draft.MOUSE.VIEW_AUTO_RELEASE_MS = Number(value); }))
                            }} width="50px" />
                        <a>&nbsp;ms</a>
                    </Grid>
                    {/* --- [修改 V1.3.5] P5 结束 --- */}
                    
                    
                    {/* --- [新增 V1.3.0] 视角重置半径 --- */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 1 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            启用视角重置半径
                        </Typography>
                        <Switch
                            checked={config.MOUSE.VIEW_RESET_RADIUS_ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.MOUSE.VIEW_RESET_RADIUS_ENABLE = !draft.MOUSE.VIEW_RESET_RADIUS_ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="视角重置半径 (%)"
                        disabled={!config.MOUSE.VIEW_RESET_RADIUS_ENABLE}
                        value={config.MOUSE.VIEW_RESET_RADIUS * 100} // %
                        min={1} max={50} step={0.5}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.MOUSE.VIEW_RESET_RADIUS = Number(value) / 100; }))
                        }}
                    />

                    {/* --- [新增 V1.3.5] P2: 重置圆环厚度 --- */}
                    <SliderWithInput
                        label="重置圆环厚度 (%)"
                        disabled={!config.MOUSE.VIEW_RESET_RADIUS_ENABLE}
                        value={config.MOUSE.VIEW_RESET_RADIUS_THICKNESS * 100} // %
                        min={0.1} max={5} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.MOUSE.VIEW_RESET_RADIUS_THICKNESS = Number(value) / 100; }))
                        }}
                    />
                    {/* --- [新增 V1.3.5] P2 结束 --- */}


                    {/* --- [新增 V1.3.0] 随机视角重置范围 --- */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 1 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            随机视角重置范围
                        </Typography>
                        <Switch
                            checked={config.MOUSE.VIEW_RANDOM_RESET_ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.MOUSE.VIEW_RANDOM_RESET_ENABLE = !draft.MOUSE.VIEW_RANDOM_RESET_ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="随机重置半径 (%)"
                        disabled={!config.MOUSE.VIEW_RANDOM_RESET_ENABLE}
                        value={config.MOUSE.VIEW_RANDOM_RESET_RADIUS * 100} // %
                        min={0.1} max={5} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.MOUSE.VIEW_RANDOM_RESET_RADIUS = Number(value) / 100; }))
                        }}
                    />
                </Paper>
                {/* --- [修改 V1.3.5] 视角设置结束 --- */}


                <Grid
                    container
                    direction="row"
                    justifyContent="flex-start"
                    alignItems="center"
                    sx={{
                        // height: "50px", // 移除固定高度
                    }}
                    spacing={1}
                >
                    <Grid item xs={12}>

                        <Typography
                            sx={{
                                width: "100%",
                                marginTop: "10px",
                            }}
                        >
                            映射切换按键：
                        </Typography>
                    </Grid>
                    {config["MOUSE"]["SWITCH_KEYS"].map((key, index) => {
                        return <Grid item key={index}>
                            <Button
                                key={index}
                                onClick={() => {
                                    setConfig(produce(draft => {
                                        draft.MOUSE.SWITCH_KEYS.splice(index, 1)
                                    }))
                                }}
                                variant="outlined"
                                sx={{
                                    width: "100%",
                                }}
                            ><Typography noWrap>{key}</Typography>
                                <HighlightOffIcon />
                            </Button>
                        </Grid>
                    })}
                    <Grid item key={"添加切换按键"}>
                        <Button
                            key={"添加切换按键"}
                            onClick={() => {
                                addingSwitchKey.current = true;
                                setAddingSwitchKeyInfoText("按下按键以添加")
                            }}
                            variant="contained"
                            sx={{
                                width: "100%",
                                // marginTop: "10px",
                            }}
                        ><Typography noWrap>{addingSwitchKeyInfoText}</Typography>
                        </Button>
                    </Grid>

                </Grid>




                <Grid
                    container
                    direction="row"
                    justifyContent="flex-start"
                    alignItems="center"
                    sx={{
                        height: "50px",
                    }}>
                    <a>{`左摇杆中心位置:(${parseInt(config["WHEEL"]["POS"][0] * config["SCREEN"]["SIZE"][0])} , ${parseInt(config["WHEEL"]["POS"][1] * config["SCREEN"]["SIZE"][1])})`} </a>
                    <Button onClick={readyToSetPos} disabled={setPosButtonDisabled || scrollSliderPosSelecting.current} sx={{ height: "30px", marginLeft: "10px" }} variant="outlined"  >重设</Button>

                </Grid>

                <Grid
                    container
                    direction="row"
                    justifyContent="flex-start"
                    alignItems="center"
                    sx={{
                        // [修改 V1.2.0] 增加高度以容纳新开关
                        height: "170px",
                    }}>

                    <Typography gutterBottom>
                        半径
                    </Typography>
                    <Grid container spacing={2}>
                        <Grid item xs>
                            <Slider
                                min={0}
                                max={50}
                                step={1}
                                value={range}
                                onChange={(_, value) => { setRange(value) }}
                                onChangeCommitted={(_, value) => {
                                    setRange(value)
                                    setConfig(produce(draft => { draft.WHEEL.RANGE = Number(value) / 100; }))
                                    if (value > shiftRange) {
                                        setShiftRange(value)
                                        setConfig(produce(draft => { draft.WHEEL.SHIFT_RANGE = Number(value) / 100; }))
                                    }
                                }}
                            />
                        </Grid>
                    </Grid>
                    <Typography gutterBottom>
                        {config["WHEEL"]["SHIFT_RANGE_ENABLE"] ? "启用shift轮盘" : "禁用shift轮盘"}
                    </Typography>
                    <Switch
                        checked={config["WHEEL"]["SHIFT_RANGE_ENABLE"]}
                        onChange={() => {
                            setConfig(produce(draft => { draft.WHEEL.SHIFT_RANGE_ENABLE = !draft.WHEEL.SHIFT_RANGE_ENABLE; }))
                        }}
                    />

                    {/* --- [修改 V1.2.0] 新 Shift 逻辑 --- */}
                    <FormControlLabel
                        control={<Switch
                            checked={config.WHEEL.SHIFT_PRESS_TOGGLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.SHIFT_PRESS_TOGGLE = !draft.WHEEL.SHIFT_PRESS_TOGGLE; }))
                            }}
                            disabled={!config.WHEEL.SHIFT_RANGE_ENABLE}
                            size="small"
                            sx={{ ml: 2 }}
                        />}
                        label={<Typography variant="body2" sx={{ fontSize: "0.9rem" }}>摁下切换</Typography>}
                    />
                    <FormControlLabel
                        control={<Switch
                            checked={config.WHEEL.SHIFT_RELEASE_TOGGLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.SHIFT_RELEASE_TOGGLE = !draft.WHEEL.SHIFT_RELEASE_TOGGLE; }))
                            }}
                            disabled={!config.WHEEL.SHIFT_RANGE_ENABLE}
                            size="small"
                            sx={{ ml: 2 }}
                        />}
                        label={<Typography variant="body2" sx={{ fontSize: "0.9rem" }}>抬起切换</Typography>}
                    />
                    {/* --- [修改 V1.2.0] 移除旧的 SHIFT_RANGE_SWITCH_ENABLE --- */}

                    <Grid container spacing={2}>
                        <Grid item xs>
                            <Slider
                                min={range}
                                max={50}
                                step={1}
                                value={shiftRange}
                                onChange={(_, value) => { setShiftRange(value) }}
                                onChangeCommitted={(_, value) => {
                                    setShiftRange(value)
                                    setConfig(produce(draft => { draft.WHEEL.SHIFT_RANGE = Number(value) / 100; }))
                                }}
                            />
                        </Grid>
                    </Grid>

                </Grid>

                {/* --- [修改 V1.2.3] 轮盘高级设置 (重构) --- */}
                <Paper sx={{ width: "100%", p: 1, marginTop: "10px" }}>
                    <Typography variant="subtitle1" gutterBottom>
                        轮盘高级设置 (V1.2.5)
                    </Typography>

                    {/* --- [修改 V1.2.3] 移除 PATH_JITTER --- */}

                    {/* 轮盘移动平滑度 */}
                    <SliderWithInput
                        label="移动平滑度"
                        value={config.WHEEL.STEP_SPEED}
                        min={1} max={120} step={1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.STEP_SPEED = Number(value); }))
                        }}
                    />
                    <Typography variant="caption" sx={{ marginLeft: "10px" }}>(原版 60, 值越小越平滑)</Typography>


                    {/* --- [修改 V1.2.3] 恒星动态速度 (重命名) --- */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 2 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            恒星动态速度
                        </Typography>
                        <Switch
                            checked={config.WHEEL.STAR_DYNAMIC_SPEED.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.STAR_DYNAMIC_SPEED.ENABLE = !draft.WHEEL.STAR_DYNAMIC_SPEED.ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="最慢速度"
                        disabled={!config.WHEEL.STAR_DYNAMIC_SPEED.ENABLE}
                        value={config.WHEEL.STAR_DYNAMIC_SPEED.MIN_SPEED}
                        min={0.1}
                        max={config.WHEEL.STEP_SPEED} // 动态绑定最大值
                        step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.STAR_DYNAMIC_SPEED.MIN_SPEED = Number(value); }))
                        }}
                    />
                    <SliderWithInput
                        label="速度周期频率"
                        disabled={!config.WHEEL.STAR_DYNAMIC_SPEED.ENABLE}
                        value={config.WHEEL.STAR_DYNAMIC_SPEED.FREQUENCY} // Hz
                        min={0.1} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.STAR_DYNAMIC_SPEED.FREQUENCY = Number(value); }))
                        }}
                    />
                    {/* --- [修改 V1.2.3] 恒星动态速度 结束 --- */}


                    {/* --- [新增 V1.2.1] 独立随机落点 (保留) --- */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 2 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            随机落点
                        </Typography>
                        <Switch
                            checked={config.WHEEL.RANDOM_START.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.RANDOM_START.ENABLE = !draft.WHEEL.RANDOM_START.ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="落点半径 (%)"
                        disabled={!config.WHEEL.RANDOM_START.ENABLE}
                        value={config.WHEEL.RANDOM_START.RADIUS * 100} // %
                        min={0} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.RANDOM_START.RADIUS = Number(value) / 100; }))
                        }}
                    />
                    {/* --- [新增 V1.2.1] 独立随机落点 结束 --- */}

                    {/* --- [修改 V1.2.3] 移除 JITTER_SMOOTH_SPEED --- */}


                    {/* --- [修改 V1.2.5] 传入 freqMax=30 --- */}
                    <CurveSettings curveType="STAR_CURVE" freqMax={30} />

                    {/* 行星转圈 */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 2 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            行星转圈 (Planet)
                        </Typography>
                        <Switch
                            checked={config.WHEEL.WHEEL_PLANET.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.ENABLE = !draft.WHEEL.WHEEL_PLANET.ENABLE; }))
                            }}
                        />
                    </Grid>
                    <SliderWithInput
                        label="行星半径"
                        disabled={!config.WHEEL.WHEEL_PLANET.ENABLE}
                        value={config.WHEEL.WHEEL_PLANET.RADIUS * 100} // %
                        min={0} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.RADIUS = Number(value) / 100; }))
                        }}
                    />
                    <SliderWithInput
                        label="行星速度"
                        disabled={!config.WHEEL.WHEEL_PLANET.ENABLE}
                        value={config.WHEEL.WHEEL_PLANET.SPEED}
                        min={0.1} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.SPEED = Number(value); }))
                        }}
                    />

                    {/* --- [新增 V1.2.0] 行星动态速度 (保留) --- */}
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 2, ml: 2 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            行星动态速度
                        </Typography>
                        <Switch
                            checked={config.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.ENABLE = !draft.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.ENABLE; }))
                            }}
                            disabled={!config.WHEEL.WHEEL_PLANET.ENABLE}
                        />
                    </Grid>
                    <SliderWithInput
                        label="最慢速度"
                        disabled={!config.WHEEL.WHEEL_PLANET.ENABLE || !config.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.ENABLE}
                        value={config.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.MIN_SPEED}
                        min={0.1}
                        max={config.WHEEL.WHEEL_PLANET.SPEED} // 动态绑定最大值
                        step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.MIN_SPEED = Number(value); }))
                        }}
                    />
                    <SliderWithInput
                        label="速度周期频率"
                        disabled={!config.WHEEL.WHEEL_PLANET.ENABLE || !config.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.ENABLE}
                        value={config.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.FREQUENCY} // Hz
                        min={0.1} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED.FREQUENCY = Number(value); }))
                        }}
                    />
                    {/* --- [新增 V1.2.0] 行星动态速度 结束 --- */}


                    {/* --- [修改 V1.2.5] 传入 freqMax=10 --- */}
                    <CurveSettings curveType="PLANET_CURVE" freqMax={10} />

                </Paper>
                {/* --- [修改 V1.2.3] 轮盘高级设置结束 --- */}


                {/* --- [新增 V1.3.0] 滚轮滑块设置 --- */}
                <Paper sx={{ width: "100%", p: 1, marginTop: "10px" }}>
                    <Grid container direction="row" justifyContent="space-between" alignItems="center">
                        <Typography variant="subtitle1" gutterBottom>
                            滚轮滑块设置
                        </Typography>
                        <Switch
                            checked={config.SCROLL_SLIDER.ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.SCROLL_SLIDER.ENABLE = !draft.SCROLL_SLIDER.ENABLE; }))
                            }}
                        />
                    </Grid>
                    <Grid
                        container
                        direction="row"
                        justifyContent="flex-start"
                        alignItems="center"
                        sx={{ height: "50px" }}
                    >
                        <a>{`中心位置:(${parseInt(config.SCROLL_SLIDER.POS[0] * config.SCREEN.SIZE[0])} , ${parseInt(config.SCROLL_SLIDER.POS[1] * config.SCREEN.SIZE[1])})`} </a>
                        <Button 
                            onClick={readyToSetScrollSliderPos} 
                            disabled={scrollSliderSetPosButtonDisabled || setPosButtonDisabled || viewCenterSetting.current} 
                            sx={{ height: "30px", marginLeft: "10px" }} 
                            variant="outlined"
                        >
                            重设
                        </Button>
                    </Grid>
                    <SliderWithInput
                        label="上滑长度 (%)"
                        disabled={!config.SCROLL_SLIDER.ENABLE}
                        value={config.SCROLL_SLIDER.LENGTH_UP * 100} // %
                        min={1} max={50} step={0.5}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.SCROLL_SLIDER.LENGTH_UP = Number(value) / 100; }))
                        }}
                    />
                    <SliderWithInput
                        label="下滑长度 (%)"
                        disabled={!config.SCROLL_SLIDER.ENABLE}
                        value={config.SCROLL_SLIDER.LENGTH_DOWN * 100} // %
                        min={1} max={50} step={0.5}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.SCROLL_SLIDER.LENGTH_DOWN = Number(value) / 100; }))
                        }}
                    />
                    <Grid
                        container
                        direction="row"
                        justifyContent="flex-start"
                        alignItems="center"
                        sx={{ height: "50px" }}
                    >
                        <a>非重置时间:</a>
                        <CostumedInput
                            key={config.SCROLL_SLIDER.TIMEOUT_S} // 强制重渲染
                            defaultValue={config.SCROLL_SLIDER.TIMEOUT_S}
                            onCommit={(value) => {
                                setConfig(produce(draft => { draft.SCROLL_SLIDER.TIMEOUT_S = Number(value); }))
                            }} width="50px" />
                        <a>&nbsp;s</a>
                    </Grid>
                    <SliderWithInput
                        label="滚动速度"
                        disabled={!config.SCROLL_SLIDER.ENABLE}
                        value={config.SCROLL_SLIDER.SPEED}
                        min={0.1} max={10} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.SCROLL_SLIDER.SPEED = Number(value); }))
                        }}
                    />
                    <Grid container direction="row" justifyContent="flex-start" alignItems="center" sx={{ mt: 1 }}>
                        <Typography gutterBottom sx={{ minWidth: "120px" }}>
                            随机范围出现
                        </Typography>
                        <Switch
                            checked={config.SCROLL_SLIDER.RANDOM_START_ENABLE}
                            onChange={() => {
                                setConfig(produce(draft => { draft.SCROLL_SLIDER.RANDOM_START_ENABLE = !draft.SCROLL_SLIDER.RANDOM_START_ENABLE; }))
                            }}
                            disabled={!config.SCROLL_SLIDER.ENABLE}
                        />
                    </Grid>
                    <SliderWithInput
                        label="随机半径 (%)"
                        disabled={!config.SCROLL_SLIDER.ENABLE || !config.SCROLL_SLIDER.RANDOM_START_ENABLE}
                        value={config.SCROLL_SLIDER.RANDOM_START_RADIUS * 100} // %
                        min={0.1} max={5} step={0.1}
                        onChange={(value) => {
                            setConfig(produce(draft => { draft.SCROLL_SLIDER.RANDOM_START_RADIUS = Number(value) / 100; }))
                        }}
                    />
                    {/* [V1.3.3] 滚轮滑块的曲线设置 (已修复) */}
                    <CurveSettings 
                        curveType="CURVE" 
                        freqMax={10} 
                        configPath="SCROLL_SLIDER" // 传入 SCROLL_SLIDER 作为路径
                    />

                </Paper>
                {/* --- [新增 V1.3.0] 滚轮滑块设置结束 --- */}


            </Grid>
        </Paper>
    }


    const Type_click = ({ data }) => {
        return <div>
            <a>点击时间 : </a>
            <CostumedInput defaultValue={data["INTERVAL"][0]} onCommit={(value) => {
                setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].INTERVAL = [Number(value)] }))
            }} />
            <a> ms</a>
        </div>
    }

    const Type_auto_fire = ({ data }) => {
        return <div>
            <a>点击时长 : </a>
            <CostumedInput defaultValue={data["INTERVAL"][0]} onCommit={(value) => {
                setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].INTERVAL[0] = Number(value) }))

            }} />
            <a> ms</a>

            <a> &emsp;间隔 : </a>
            <CostumedInput defaultValue={data["INTERVAL"][1]} onCommit={(value) => {
                setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].INTERVAL[1] = Number(value) }))
            }} />
            <a> ms</a>
        </div>
    }

    const Type_drag = ({ data }) => {
        const waitingForClick = useRef(false)
        const [addButtonDisabled, setAddButtonDisabled] = useState(false)
        const readyToAdd = () => { waitingForClick.current = true; setAddButtonDisabled(true) }
        const addKeyPoint = (x, y) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.push([x, y]) }))
        }

        const removeKeyPoint = (index) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.splice(index, 1) }))
        }

        const imgClickListener = (e) => {
            if (waitingForClick.current) {
                addKeyPoint(e.detail.x, e.detail.y)
                waitingForClick.current = false;
                setAddButtonDisabled(false)
            }
        }
        useEffect(() => {
            window.addEventListener('imgOnNoKeyClick', imgClickListener)
            return () => {
                window.removeEventListener('imgOnNoKeyClick', imgClickListener)
            }
        }, [])

        return <div>
            <Grid container >
                <Grid item xs={6}><a>间隔 : </a>
                    <CostumedInput defaultValue={config.KEY_MAPS[data["KEY"]].INTERVAL[0]} onCommit={(value) => {
                        setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].INTERVAL = [Number(value)] }))
                    }} />
                    <a> ms </a></Grid>
                <Grid item xs={6}><Button onClick={readyToAdd} disabled={addButtonDisabled} variant="outlined" sx={{
                    height: "30px",
                    width: "105px",
                }}  >添加关键点</Button></Grid>
            </Grid>
            {
                data["POS_S"].map((pos, index) => <div key={index} style={{ display: "flex" }}>
                    <a>{index}&emsp;{`(${getDisplayValueX(pos[0])} , ${getDisplayValueY(pos[1])})`}</a>
                    <IconButton onClick={() => { removeKeyPoint(index) }}>
                        <HighlightOffIcon />
                    </IconButton>
                </div>
                )
            }
        </div>
    }


    const Type_mult_press = ({ data }) => {
        const waitingForClick = useRef(false)
        const [addButtonDisabled, setAddButtonDisabled] = useState(false)
        const readyToAdd = () => { waitingForClick.current = true; setAddButtonDisabled(true) }

        const addKeyPoint = (x, y) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.push([x, y]) }))
        }

        const removeKeyPoint = (index) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.splice(index, 1) }))
        }

        const imgClickListener = (e) => {
            if (waitingForClick.current) {
                console.log("imgClickListener", e.detail);
                addKeyPoint(e.detail.x, e.detail.y)
                waitingForClick.current = false;
                setAddButtonDisabled(false)
            }
        }
        useEffect(() => {
            window.addEventListener('imgOnNoKeyClick', imgClickListener)
            return () => {
                window.removeEventListener('imgOnNoKeyClick', imgClickListener)
            }
        }, [])

        return <div>
            <Grid container >
                <Grid item xs={6}><Button onClick={readyToAdd} disabled={addButtonDisabled} variant="outlined" sx={{
                    height: "30px",
                    width: "105px",
                }}  >添加触摸点</Button></Grid>
            </Grid>
            {
                data["POS_S"].map((pos, index) => <div key={index} style={{ display: "flex" }}>
                    <a>{index}&emsp;{`(${getDisplayValueX(pos[0])} , ${getDisplayValueY(pos[1])})`}</a>
                    <IconButton onClick={() => { removeKeyPoint(index) }}>
                        <HighlightOffIcon />
                    </IconButton>
                </div>
                )
            }
        </div>
    }

    // --- [新增 V1.3.0] 背包键 UI ---
    const Type_backpack_toggle = ({ data }) => {
        const waitingForClick = useRef(false)
        const [addButtonDisabled, setAddButtonDisabled] = useState(false)
        const readyToAdd = () => { waitingForClick.current = true; setAddButtonDisabled(true) }

        const setKeyPoint = (x, y) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_B = [x, y] }))
        }

        const imgClickListener = (e) => {
            if (waitingForClick.current) {
                setKeyPoint(e.detail.x, e.detail.y)
                waitingForClick.current = false;
                setAddButtonDisabled(false)
            }
        }
        useEffect(() => {
            window.addEventListener('imgOnNoKeyClick', imgClickListener)
            return () => {
                window.removeEventListener('imgOnNoKeyClick', imgClickListener)
            }
        }, [])

        const posA = data.POS;
        const posB = data.POS_B;

        return <div>
            <Grid container >
                <Grid item xs={12}><Button onClick={readyToAdd} disabled={addButtonDisabled} variant="outlined" sx={{
                    height: "30px",
                    width: "150px",
                }}  >设置第二次位置</Button></Grid>
            </Grid>
            <div style={{ display: "flex" }}>
                <a>A&emsp;{`(${getDisplayValueX(posA[0])} , ${getDisplayValueY(posA[1])})`}</a>
            </div>
            <div style={{ display: "flex" }}>
                <a>B&emsp;{`(${getDisplayValueX(posB[0])} , ${getDisplayValueY(posB[1])})`}</a>
            </div>
        </div>
    }
    
    // --- [新增 V1.3.0] 依次触摸点 UI ---
    const Type_sequential_press = ({ data }) => {
        const waitingForClick = useRef(false)
        const [addButtonDisabled, setAddButtonDisabled] = useState(false)
        const readyToAdd = () => { waitingForClick.current = true; setAddButtonDisabled(true) }

        const addKeyPoint = (x, y) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.push([x, y]) }))
        }

        const removeKeyPoint = (index) => {
            setConfig(produce(draft => { draft.KEY_MAPS[data["KEY"]].POS_S.splice(index, 1) }))
        }

        const imgClickListener = (e) => {
            if (waitingForClick.current) {
                addKeyPoint(e.detail.x, e.detail.y)
                waitingForClick.current = false;
                setAddButtonDisabled(false)
            }
        }
        useEffect(() => {
            window.addEventListener('imgOnNoKeyClick', imgClickListener)
            return () => {
                window.removeEventListener('imgOnNoKeyClick', imgClickListener)
            }
        }, [])
        
        // 第一个点是按键自己的 POS
        const allPoints = [data.POS, ...(data.POS_S || [])];

        return <div>
            <Grid container >
                <Grid item xs={6}><Button onClick={readyToAdd} disabled={addButtonDisabled} variant="outlined" sx={{
                    height: "30px",
                    width: "105px",
                }}  >添加后续点</Button></Grid>
            </Grid>
            {
                allPoints.map((pos, index) => <div key={index} style={{ display: "flex" }}>
                    <a>{index+1}&emsp;{`(${getDisplayValueX(pos[0])} , ${getDisplayValueY(pos[1])})`}</a>
                    {/* 第一个点 (index 0) 不允许删除 */}
                    {index > 0 && <IconButton onClick={() => { removeKeyPoint(index - 1) }}> 
                        <HighlightOffIcon />
                    </IconButton>}
                </div>
                )
            }
        </div>
    }
    // --- [新增 V1.3.0] 结束 ---


    const KeySettingRender = ({ data }) => {
        const isWheel = data["KEY"] === "REL_WHEEL_UP" || data["KEY"] === "REL_WHEEL_DOWN"


        const handleChange = (e) => {
            // [V1.3.0] 提取当前位置 (或设置默认)
            let currentPos = [0.4, 0.4];
            if (Object.keys(config["KEY_MAPS"][data["KEY"]]).indexOf("POS") !== -1) {
                currentPos = config["KEY_MAPS"][data["KEY"]]["POS"];
            }

            if (e.target.value === "CLICK") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "CLICK", "POS": currentPos, "INTERVAL": [18] }
                }))
            } else if (e.target.value === "PRESS") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "PRESS", "POS": currentPos }
                }))
            } else if (e.target.value === "AUTO_FIRE") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "AUTO_FIRE", "POS": currentPos, "INTERVAL": [18, 20] }
                }))
            } else if (e.target.value === "DRAG") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "DRAG", "POS_S": [], "INTERVAL": [18] }
                }))
            } else if (e.target.value === "MULT_PRESS") {
                setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "MULT_PRESS", "POS_S": [], }
                }))
            // --- [新增 V1.3.0] 新按键类型 ---
            } else if (e.target.value === "SYNC_VIEW_RESET") {
                 setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "SYNC_VIEW_RESET", "POS": currentPos }
                }))
            } else if (e.target.value === "CLICK_VIEW_RESET") {
                 setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "CLICK_VIEW_RESET", "POS": currentPos }
                }))
            } else if (e.target.value === "BACKPACK_TOGGLE") {
                 setConfig(produce(draft => {
                    // 背包键需要两个位置, A (当前) 和 B (默认 B = A)
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "BACKPACK_TOGGLE", "POS": currentPos, "POS_B": currentPos }
                }))
            } else if (e.target.value === "CLICK_MAP_ON") {
                 setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "CLICK_MAP_ON", "POS": currentPos, "INTERVAL": [18] } // 基于 CLICK
                }))
            } else if (e.target.value === "CLICK_MAP_OFF") {
                 setConfig(produce(draft => {
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "CLICK_MAP_OFF", "POS": currentPos, "INTERVAL": [18] } // 基于 CLICK
                }))
            } else if (e.target.value === "SEQUENTIAL_PRESS") {
                 setConfig(produce(draft => {
                    // 依次触摸点需要一个 POS 数组, 第一个点是按键的 POS
                    draft.KEY_MAPS[data["KEY"]] = { "TYPE": "SEQUENTIAL_PRESS", "POS": currentPos, "POS_S": [] }
                }))
            }
            // --- [新增 V1.3.0] 结束 ---
        }
        
        // [V1.3.0] 扩展需要显示坐标的类型
        const posBasedTypes = ["PRESS", "AUTO_FIRE", "CLICK", "SYNC_VIEW_RESET", "CLICK_VIEW_RESET", "BACKPACK_TOGGLE", "CLICK_MAP_ON", "CLICK_MAP_OFF", "SEQUENTIAL_PRESS"];
        const showPos = posBasedTypes.includes(data["TYPE"]);

        return <Grid
            container
            direction="column"
            padding="10px"
        >
            <Grid
                container
                direction="row"
                justifyContent="flex-start"
                alignItems="center"
            >
                {
                    showPos ?
                        <Grid item xs={5}><a>{`${data["KEY"]} : (${getDisplayValueX(data["POS"][0])} , ${getDisplayValueY(data["POS"][1])})`}</a></Grid> :
                        <Grid item xs={5}><a>{`${data["KEY"]} `}</a></Grid>
                }
                <Grid item xs={5}>
                    <FormControl>
                        <InputLabel id={`${data["KEY"]}-select`}></InputLabel>
                        <Select
                            labelId={`${data["KEY"]}-select-label`}
                            value={data["TYPE"]}
                            onChange={handleChange}
                            sx={{ height: "30px", }}
                        >
                            {!isWheel && <MenuItem value={"PRESS"}>同步按下释放</MenuItem>}
                            <MenuItem value={"CLICK"}>单次点击</MenuItem>
                            {!isWheel && <MenuItem value={"AUTO_FIRE"}>连发</MenuItem>}
                            <MenuItem value={"DRAG"}>滑动</MenuItem>
                            {!isWheel && <MenuItem value={"MULT_PRESS"}>多点触摸</MenuItem>}
                            {/* --- [新增 V1.3.0] 新按键类型 --- */}
                            {!isWheel && <MenuItem value={"SYNC_VIEW_RESET"}>同步按抬鼠标重置</MenuItem>}
                            {!isWheel && <MenuItem value={"CLICK_VIEW_RESET"}>单点击鼠标重置</MenuItem>}
                            {!isWheel && <MenuItem value={"BACKPACK_TOGGLE"}>背包键</MenuItem>}
                            {!isWheel && <MenuItem value={"CLICK_MAP_ON"}>开启映射后点击</MenuItem>}
                            {!isWheel && <MenuItem value={"CLICK_MAP_OFF"}>点击关闭映射</MenuItem>}
                            {!isWheel && <MenuItem value={"SEQUENTIAL_PRESS"}>依次触摸点</MenuItem>}
                            {/* --- [新增 V1.3.0] 结束 --- */}
                        </Select>
                    </FormControl>
                </Grid>
                <Grid item xs={2}>
                    <IconButton onClick={() => {
                        setConfig(produce(draft => { delete draft.KEY_MAPS[data["KEY"]] }))
                    }}>
                        <HighlightOffIcon />
                    </IconButton>
                </Grid>
            </Grid>
            {data["TYPE"] === "CLICK" ? <Type_click data={data} /> : null}
            {data["TYPE"] === "AUTO_FIRE" ? <Type_auto_fire data={data} /> : null}
            {data["TYPE"] === "DRAG" ? <Type_drag data={data} /> : null}
            {data["TYPE"] === "MULT_PRESS" ? <Type_mult_press data={data} /> : null}
            {/* --- [新增 V1.3.0] 新按键类型 UI 渲染 --- */}
            {data["TYPE"] === "BACKPACK_TOGGLE" ? <Type_backpack_toggle data={data} /> : null}
            {data["TYPE"] === "SEQUENTIAL_PRESS" ? <Type_sequential_press data={data} /> : null}
            {/* [V1.3.0] 其他4个新类型不需要额外UI */}
            {/* --- [新增 V1.3.0] 结束 --- */}

        </Grid>
    }


    // --- [修改 V1.3.5] 修复 P2 和 P5 的加载逻辑 ---
    useEffect(() => {
        document.onkeydown = (e) => {
            if (e.repeat === false && window.stopPreventDefault !== true) {
                e.preventDefault();
                if (addingSwitchKey.current) {
                    setConfig(produce(draft => {
                        if (draft.MOUSE.SWITCH_KEYS.indexOf(keyNameMap[e.code.toLowerCase()]) === -1) {
                            draft.MOUSE.SWITCH_KEYS.push(keyNameMap[e.code.toLowerCase()])
                            setAddingSwitchKeyInfoText("添加映射切换键")
                        } else {
                            setAddingSwitchKeyInfoText("已存在，请重新添加")

                        }
                    }))
                } else {
                    setSelectKEY(keyNameMap[e.code.toLowerCase()])
                }
            }
        }
        document.onkeyup = (e) => {
            if (window.stopPreventDefault !== true) {
                e.preventDefault();
                setSelectKEY(null)
            }
        }
        document.oncontextmenu = function (e) {
            e.preventDefault();
        };

        window.addEventListener("resize", (e) => {
            if (document.getElementById("img")) {
                setImgSize([document.getElementById("img").width, document.getElementById("img").height])
            }
        })

        // --- [修复 V1.3.2] 将这两个变量添加回来 ---
        // V1.2.3 默认 "曲线" 配置
        const defaultStarCurve = {
            ENABLE: false,
            CURVE_AMOUNT: 0.002,
            CURVE_FREQUENCY: 1.0,
        };
        const defaultPlanetCurve = {
            ENABLE: false,
            CURVE_AMOUNT: 0.005,
            CURVE_FREQUENCY: 1.0,
        };
        // --- [修复 V1.3.2] 结束 ---
        
        // [V1.3.5] 滚轮滑块默认 (扁平)
        const defaultScrollSlider = {
            "ENABLE": false,
            "POS": [ 0.9, 0.5 ],
            "LENGTH_UP": 0.2,
            "LENGTH_DOWN": 0.2,
            "TIMEOUT_S": 3,
            "SPEED": 1.0,
            "RANDOM_START_ENABLE": false,
            "RANDOM_START_RADIUS": 0.005,
            "CURVE_ENABLE": false,
            "CURVE_AMOUNT": 0.005
        }


        // [V1.3.5] 默认配置检查器
        const safeCheckConfig = (data) => {
            if (!data.MOUSE) data.MOUSE = {};
            // --- [修改 V1.3.5] P5: 自动释放加载逻辑 ---
            if (data.MOUSE.VIEW_AUTO_RELEASE_ENABLE === undefined) {
                // 迁移旧逻辑: 如果 MS 是 0, 说明是关闭
                if (data.MOUSE.VIEW_AUTO_RELEASE_MS === 0) {
                    data.MOUSE.VIEW_AUTO_RELEASE_ENABLE = false;
                    data.MOUSE.VIEW_AUTO_RELEASE_MS = 200; // 重置为默认
                } else if (data.MOUSE.VIEW_AUTO_RELEASE_MS > 0) {
                     data.MOUSE.VIEW_AUTO_RELEASE_ENABLE = true; // 启用
                } else {
                    data.MOUSE.VIEW_AUTO_RELEASE_ENABLE = false; // 默认关闭
                }
            }
            if (data.MOUSE.VIEW_AUTO_RELEASE_MS === undefined || data.MOUSE.VIEW_AUTO_RELEASE_MS === 0) {
                 data.MOUSE.VIEW_AUTO_RELEASE_MS = 200; // 默认 200
            }
            // --- [修改 V1.3.5] P5 结束 ---
            
            if (data.MOUSE.VIEW_RESET_RADIUS_ENABLE === undefined) data.MOUSE.VIEW_RESET_RADIUS_ENABLE = false;
            if (data.MOUSE.VIEW_RESET_RADIUS === undefined) data.MOUSE.VIEW_RESET_RADIUS = 0.1;
            
            // --- [新增 V1.3.5] P2: 加载圆环厚度 ---
            if (data.MOUSE.VIEW_RESET_RADIUS_THICKNESS === undefined) data.MOUSE.VIEW_RESET_RADIUS_THICKNESS = 0.005;
            // --- [新增 V1.3.5] P2 结束 ---

            if (data.MOUSE.VIEW_RANDOM_RESET_ENABLE === undefined) data.MOUSE.VIEW_RANDOM_RESET_ENABLE = false;
            if (data.MOUSE.VIEW_RANDOM_RESET_RADIUS === undefined) data.MOUSE.VIEW_RANDOM_RESET_RADIUS = 0.01;


            if (!data.WHEEL) data.WHEEL = {};

            // 检查曲线 (Curve)
            if (!data.WHEEL.STAR_CURVE) {
                data.WHEEL.STAR_CURVE = { ...defaultStarCurve };
            }
            if (!data.WHEEL.PLANET_CURVE) {
                data.WHEEL.PLANET_CURVE = { ...defaultPlanetCurve };
            }

            // 检查行星 (Planet)
            if (!data.WHEEL.WHEEL_PLANET) {
                data.WHEEL.WHEEL_PLANET = { ENABLE: false, RADIUS: 0.015, SPEED: 1.5 };
            }
            if (!data.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED) {
                data.WHEEL.WHEEL_PLANET.PLANET_DYNAMIC_SPEED = { ENABLE: false, MIN_SPEED: 0.5, FREQUENCY: 1.0 };
            }

            // 检查恒星动态速度 V1.2.3
            if (!data.WHEEL.STAR_DYNAMIC_SPEED) {
                data.WHEEL.STAR_DYNAMIC_SPEED = { ENABLE: false, MIN_SPEED: 10.0, FREQUENCY: 1.0 };
            }
            if (data.WHEEL.STEP_SPEED === undefined) data.WHEEL.STEP_SPEED = 60;

            // 检查新 Shift 逻辑 V1.2.0
            if (data.WHEEL.SHIFT_PRESS_TOGGLE === undefined) data.WHEEL.SHIFT_PRESS_TOGGLE = false;
            if (data.WHEEL.SHIFT_RELEASE_TOGGLE === undefined) data.WHEEL.SHIFT_RELEASE_TOGGLE = false;

            // 检查随机落点 V1.2.1
            if (!data.WHEEL.RANDOM_START) {
                data.WHEEL.RANDOM_START = { ENABLE: false, RADIUS: 0.01 };
            }
            
            // [V1.3.3] 检查滚轮滑块 (扁平结构)
            if (!data.SCROLL_SLIDER) {
                data.SCROLL_SLIDER = { ...defaultScrollSlider };
            }
            // [V1.3.3] 修复: 检查扁平的 CURVE_ENABLE 和 CURVE_AMOUNT
            if (data.SCROLL_SLIDER.CURVE_ENABLE === undefined) {
                data.SCROLL_SLIDER.CURVE_ENABLE = defaultScrollSlider.CURVE_ENABLE;
            }
            if (data.SCROLL_SLIDER.CURVE_AMOUNT === undefined) {
                 data.SCROLL_SLIDER.CURVE_AMOUNT = defaultScrollSlider.CURVE_AMOUNT;
            }
            // [V1.3.3] 确保所有 SCROLL_SLIDER 字段都存在
            if (data.SCROLL_SLIDER.POS === undefined) data.SCROLL_SLIDER.POS = defaultScrollSlider.POS;
            if (data.SCROLL_SLIDER.LENGTH_UP === undefined) data.SCROLL_SLIDER.LENGTH_UP = defaultScrollSlider.LENGTH_UP;
            if (data.SCROLL_SLIDER.LENGTH_DOWN === undefined) data.SCROLL_SLIDER.LENGTH_DOWN = defaultScrollSlider.LENGTH_DOWN;
            if (data.SCROLL_SLIDER.TIMEOUT_S === undefined) data.SCROLL_SLIDER.TIMEOUT_S = defaultScrollSlider.TIMEOUT_S;
            if (data.SCROLL_SLIDER.SPEED === undefined) data.SCROLL_SLIDER.SPEED = defaultScrollSlider.SPEED;
            if (data.SCROLL_SLIDER.RANDOM_START_ENABLE === undefined) data.SCROLL_SLIDER.RANDOM_START_ENABLE = defaultScrollSlider.RANDOM_START_ENABLE;
            if (data.SCROLL_SLIDER.RANDOM_START_RADIUS === undefined) data.SCROLL_SLIDER.RANDOM_START_RADIUS = defaultScrollSlider.RANDOM_START_RADIUS;


            // 检查按键抖动 (Key Jitter)
            if (!data.KEY_JITTER) {
                data.KEY_JITTER = { ENABLE: true, AMOUNT: 0.003 };
            }

            // --- 移除所有旧的/不兼容的键 ---
            delete data.WHEEL.PATH_JITTER;
            delete data.WHEEL.STAR_JITTER;
            delete data.WHEEL.PLANET_JITTER;
            delete data.WHEEL.JITTER_SMOOTH_SPEED;
            delete data.WHEEL.PATH_DYNAMIC_SPEED; // V1.2.3 已重命名
            delete data.WHEEL.WHEEL_JITTER; // V1.0
            delete data.WHEEL.SHIFT_RANGE_SWITCH_ENABLE; // V1.0

            return data;
        };


        fetch("/configure/get")
            .then(resp => resp.json())
            .then(data => {
                const safeData = safeCheckConfig(data);
                setConfig(safeData);
            })
            .catch(err => {
                console.log(err)
            })
    }, [])
    // --- [修改 V1.3.5] useEffect 结束 ---

    const KeyShow = ({ data }) => {
        // [V1.3.0] 扩展需要显示坐标的类型
        const posBasedTypes = ["PRESS", "AUTO_FIRE", "CLICK", "SYNC_VIEW_RESET", "CLICK_VIEW_RESET", "BACKPACK_TOGGLE", "CLICK_MAP_ON", "CLICK_MAP_OFF", "SEQUENTIAL_PRESS"];
        const showPos = posBasedTypes.includes(data["TYPE"]);

        // [V1.3.0] 扩展多点显示
        const multiPosTypes = ["MULT_PRESS", "DRAG", "SEQUENTIAL_PRESS", "BACKPACK_TOGGLE"];
        const showMultiPos = multiPosTypes.includes(data["TYPE"]);

        // [V1.3.0] 决定显示的点
        let points = [];
        let bgColor = "#d90051"; // 默认 PRESS
        let textColor = "#ffffff";
        
        if (showPos && !showMultiPos) {
             points = [data.POS];
        } else if (data.TYPE === "MULT_PRESS") {
            points = data.POS_S || [];
            bgColor = "#00796B";
        } else if (data.TYPE === "DRAG") {
            points = data.POS_S || [];
            bgColor = "#3F51B5";
        } else if (data.TYPE === "SEQUENTIAL_PRESS") {
            points = [data.POS, ...(data.POS_S || [])];
            bgColor = "#F57C00";
        } else if (data.TYPE === "BACKPACK_TOGGLE") {
            // [V1.3.1] 修复：确保 POS_B 存在
            points = [data.POS, (data.POS_B || data.POS)];
            bgColor = "#7B1FA2";
        }

        return <div>
            {showPos && !showMultiPos ? 
                <FixedIcon x={getPostionValueX(data["POS"][0])} y={getPostionValueY(data["POS"][1])} text={data["KEY"]} /> 
                : null
            }
            {showMultiPos ? 
                <GroupFixedIcon 
                    pos_s={points.map(([x, y]) => [getPostionValueX(x), getPostionValueY(y)])} 
                    text={data["KEY"]} 
                    bgColor={bgColor} 
                    textColor={textColor} 
                /> 
                : null
            }
        </div>
    }

    return <div style={{
        width: '100vw',
        height: '100vh',
        backgroundColor: '#00796B',
    }}>
        <div>{JSON.stringify()}</div>
        <JoystickListener setDowningBtn={(value) => {
            setSelectKEY(value)
        }} />
        <input id="fileInput" type="file" style={{ display: "none" }} accept="image/*" onChange={handleFileChange} ></input>
        <img id="img" src={config["IMG"]} style={{ width: "100vw", left: 0, top: 0 }} onClick={handelImgClick} onLoad={imgLoaded} ></img>
        <DraggableContainer>
            <div
                style={{
                    maxHeight: "80vh",
                    overflowY: "scroll",
                }}
            >
                <Grid
                    container
                    direction="column"
                    justifyContent="flex-start"
                    alignItems="flex-start"
                    spacing={"10px"}
                    sx={{
                        width: "400px",
                        backgroundColor: "#F5F5F5",
                        paddingBottom: "10px",
                        spacing: "0px",
                        paddingTop: "10px",
                    }}
                >
                    <Grid item xs={12}>
                        <OtherSettings />
                    </Grid>
                    {
                        Object.keys(config["KEY_MAPS"]).map((keycode, index) =>
                            <Grid
                                item
                                xs={12}
                                key={keycode}
                            >
                                <Paper
                                    sx={{
                                        width: "370px",
                                        marginLeft: "10px",
                                    }}
                                >
                                    <KeySettingRender data={{ ...config["KEY_MAPS"][keycode], "KEY": keycode }} />
                                </Paper>
                            </Grid>)
                    }
                </Grid>
            </div>
        </DraggableContainer>

        {
            Object.keys(config["KEY_MAPS"]).map((keycode, index) => <KeyShow key={keycode} data={{ ...config["KEY_MAPS"][keycode], "KEY": keycode }} />)
        }
        <WheelShow
            x={getPostionValueX(config["WHEEL"]["POS"][0])}
            y={getPostionValueY(config["WHEEL"]["POS"][1])}
            range={getPostionValueX(config["WHEEL"]["RANGE"])}
            shift_range={config["WHEEL"]["SHIFT_RANGE_ENABLE"] ? getPostionValueX(config["WHEEL"]["SHIFT_RANGE"]) : 0}
        />
        <ViewShow x={getPostionValueX(config["MOUSE"]["POS"][0])} y={getPostionValueY(config["MOUSE"]["POS"][1])} />
        
        {/* --- [新增 V1.3.1] 渲染新组件 --- */}
        <ViewResetRadiusShow
             x={getPostionValueX(config["MOUSE"]["POS"][0])}
             y={getPostionValueY(config["MOUSE"]["POS"][1])}
             radius={getPostionValueX(config.MOUSE.VIEW_RESET_RADIUS)} 
             enable={config.MOUSE.VIEW_RESET_RADIUS_ENABLE}
        />
        <ScrollSliderShow
            x={getPostionValueX(config.SCROLL_SLIDER.POS[0])}
            y={getPostionValueY(config.SCROLL_SLIDER.POS[1])}
            lengthUp={getPostionValueY(config.SCROLL_SLIDER.LENGTH_UP)}
            lengthDown={getPostionValueY(config.SCROLL_SLIDER.LENGTH_DOWN)}
            enable={config.SCROLL_SLIDER.ENABLE}
        />
        {/* --- [新增 V1.3.1] 结束 --- */}
    </div>
}
