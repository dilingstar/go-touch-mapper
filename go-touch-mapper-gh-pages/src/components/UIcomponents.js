import { useEffect, useRef, useState } from "react";
import { Button, IconButton, Input, Paper } from "@mui/material";

const UploadButton = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: 'calc(50% - 100px)',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>上传图片</button>
}

const UploadButtonJIETU = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: '50%',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>屏幕截图</button>
}



const UploadButton5s = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: 'calc(50% + 100px)',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>5s后截图</button>
}



// const FixedIcon = (props) => {
const FixedIcon = ({ x, y, size, bgColor, textColor, text }) => {
    return <button
        style={{
            position: 'absolute',
            left: x,
            top: y,
            width: size || 28,
            height: size || 28,
            borderRadius: size || 28,
            backgroundColor: bgColor || "#d90051",
            color: textColor || "white",
            marginLeft: size / -2 || -14,
            marginTop: size / -2 || -14,
            border: "None",
            alignItems: "center",
            pointerEvents: "none",
        }}
    >
        {text}
    </button>
}
const GroupFixedIcon = ({ pos_s, bgColor, textColor, text }) => {
    return <div>
        {
            pos_s.map((pos, index) => <FixedIcon
                key={index}
                x={pos[0]}
                y={pos[1]}
                size={18}
                bgColor={bgColor}
                textColor={textColor}
                text={`${text}_${index}`}
            />)
        }
    </div>
}

const CostumedInput = ({ defaultValue, width, onCommit }) => {
    const [value, setValue] = useState(defaultValue)
    return <Input
        sx={{ width: width || "40px" }}
        // inputProps={{ inputMode: 'numeric', pattern: '[0-9]*' }}
        value={value}
        onChange={(e) => {
            setValue(e.target.value)
        }}
        onFocus={(e) => {
            window.stopPreventDefault = true
        }}
        onBlur={(e) => {
            window.stopPreventDefault = false
            onCommit && onCommit(Number(value))
        }}
        onKeyDown={(e) => {
            if (e.key === "Enter") {
                onCommit && onCommit(Number(value))
            }
        }}
    />
}

const WheelShow = ({ x, y, range, shift_range }) => {
    const radius = range * 2
    const shift_radius = shift_range * 2
    return <div>
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: 16,
            height: 16,
            borderRadius: 16,
            marginLeft: -8,
            marginTop: -8,
            backgroundColor: "#2196F3",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: radius,
            height: radius,
            borderRadius: radius,
            marginLeft: radius / -2 - 4,
            marginTop: radius / -2 - 4,
            border: "4px solid #2196F3",
            pointerEvents: "none",
        }} />
        {
            shift_range !== 0 && <div style={{
                position: 'absolute',
                left: x,
                top: y,
                width: shift_radius,
                height: shift_radius,
                borderRadius: shift_radius,
                marginLeft: shift_radius / -2 - 4,
                marginTop: shift_radius / -2 - 4,
                border: "4px solid #512DA8",
                pointerEvents: "none",
            }} />
        }
    </div>
}


const ViewShow = ({ x, y }) => {
    // 视角中心点显示
    //在屏幕上显示两条垂直线，中间是一个小的空心圆
    return <div>
        <div style={{
            position: 'absolute',
            left: 0,
            top: y,
            width: "100vw",
            height: 1,
            backgroundColor: "#d90051",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: 0,
            height: "100vh",
            width: 1,
            backgroundColor: "#d90051",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: 32,
            height: 32,
            borderRadius: 16,
            marginLeft: -16,
            marginTop: -16,
            border: "2px solid #d90051",
            backgroundColor: "#transparent",
            pointerEvents: "none",
        }} />
    </div>
}

// --- [新增 V1.3.0] 视角重置半径 可视化 ---
const ViewResetRadiusShow = ({ x, y, radius, enable }) => {
    if (!enable) {
        return null; // 如果未启用，则不渲染
    }
    const diameter = radius * 2;
    return <div style={{
        position: 'absolute',
        left: x,
        top: y,
        width: diameter,
        height: diameter,
        borderRadius: "50%", // 确保是圆形
        marginLeft: diameter / -2 - 1, // -1 是为了 (border width / 2)
        marginTop: diameter / -2 - 1,
        border: "2px dashed #FFEB3B", // 黄色虚线
        pointerEvents: "none",
        opacity: 0.7,
    }} />
}

// --- [新增 V1.3.0] 滚轮滑块 可视化 ---
const ScrollSliderShow = ({ x, y, lengthUp, lengthDown, enable }) => {
    if (!enable) {
        return null; // 如果未启用，则不渲染
    }
    const totalHeight = lengthUp + lengthDown;
    return <div>
        {/* 中心圆点 (暖色) */}
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: 12,
            height: 12,
            borderRadius: 6,
            marginLeft: -6,
            marginTop: -6,
            backgroundColor: "#FF9800", // 橙色
            pointerEvents: "none",
            zIndex: 1,
        }} />
        {/* 竖直条带 (冷色半透明) */}
        <div style={{
            position: 'absolute',
            left: x,
            top: y - lengthUp, // 从上边界开始
            width: 10, // 滑块偏小
            height: totalHeight, // 总高度
            marginLeft: -5,
            backgroundColor: "#03A9F4", // 蓝色
            opacity: 0.5, // 半透明
            pointerEvents: "none",
            borderRadius: 5,
        }} />
    </div>
}
// --- [新增 V1.3.0] 结束 ---


export {
    UploadButton,
    UploadButtonJIETU,
    UploadButton5s,
    FixedIcon,
    GroupFixedIcon,
    CostumedInput,
    WheelShow,
    ViewShow,
    ViewResetRadiusShow,
    ScrollSliderShow,
}
