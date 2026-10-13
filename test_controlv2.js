/**
 * Smart IoT Service - Control V2 模拟测试客户端
 * 
 * 功能：
 * 1. 连接 TCP 9998 端口 (ControlServiceV2)
 * 2. 接收服务端发送的读取 ID 指令 (00 03 00 60 00 04 45 C6)
 * 3. 响应设备 ID 报文: 00 03 00 60 00 04 0F 0B 19 01 [CRC16] (对应设备ID: 15112501)
 * 4. 接收服务端发送的数据轮询指令 (00 03 00 00 F0 24)
 * 5. 通道1模拟随机温度数据 (0.1精度，如 25.6℃ -> 256)
 * 6. 维护并初始化 32 个继电器状态，每次轮询将当前 32 个继电器的值填入报文
 * 7. 支持接收并响应继电器控制指令 (00 10 00 7A ...)，自动更新对应继电器状态
 * 8. 支持终端实时交互：输入 1~32 切换指定继电器，输入 r 随机重置，输入 0 全关，输入 1 全开
 *
 * 运行方式:
 *   node test_controlv2.js                     (默认连接 124.222.140.69:9998，16通道)
 *   node test_controlv2.js --relays=random     (随机初始化32个继电器)
 *   node test_controlv2.js --relays=all_on     (初始化32个继电器全部开启)
 *   node test_controlv2.js --relays=all_off    (初始化32个继电器全部关闭)
 *   node test_controlv2.js --active            (连接后主动上报一次设备ID)
 *   node test_controlv2.js --mode=64           (测试 64通道 A1 报文)
 *   node test_controlv2.js --mode=32           (测试 32通道 A0 报文)
 *   node test_controlv2.js --host=127.0.0.1    (指定本地测试)
 */

const net = require('net');

// 解析命令行参数
const ARGS = process.argv.slice(2).reduce((acc, arg) => {
  const [k, v] = arg.replace(/^--/, '').split('=');
  acc[k] = v || true;
  return acc;
}, {});

const HOST = process.env.HOST || ARGS.host || '124.222.140.69';
const PORT = parseInt(process.env.PORT || ARGS.port || 9998, 10);
const MODE = (ARGS.mode || '16').toString(); // '16', '32', '64'
const ACTIVE = Boolean(ARGS.active);
const ONCE = Boolean(ARGS.once);
const TIMEOUT = parseInt(ARGS.timeout || 0, 10);
const RELAYS_INIT_MODE = ARGS.relays || 'default'; // 'default', 'random', 'all_on', 'all_off'

// 固定测试 ID: 0F 0B 19 01 -> 15112501
const DEVICE_ID_BYTES = [0x0F, 0x0B, 0x19, 0x01];
const DEVICE_ID_NUM = (
  DEVICE_ID_BYTES[0] * 1000000 +
  DEVICE_ID_BYTES[1] * 10000 +
  DEVICE_ID_BYTES[2] * 100 +
  DEVICE_ID_BYTES[3]
);

// 32 个继电器状态数组 (索引 0~31 对应 继电器 1~32，状态 0=关闭, 1=开启)
const relays = new Uint8Array(32);

function initRelays(mode) {
  if (typeof mode === 'string' && mode.includes(',')) {
    const list = mode.split(',').map(v => parseInt(v.trim(), 10) ? 1 : 0);
    for (let i = 0; i < 32; i++) relays[i] = list[i] || 0;
  } else if (mode === 'all_on' || mode === '1') {
    relays.fill(1);
  } else if (mode === 'all_off' || mode === '0') {
    relays.fill(0);
  } else if (mode === 'random') {
    for (let i = 0; i < 32; i++) {
      relays[i] = Math.random() > 0.5 ? 1 : 0;
    }
  } else {
    // 默认模式：初始化前 8 路继电器交替开启，直观易观察
    for (let i = 0; i < 32; i++) {
      relays[i] = (i === 0 || i === 2 || i === 4 || i === 7 || i === 15) ? 1 : 0;
    }
  }
}

// 格式化继电器统计信息
function formatRelaysSummary() {
  const onList = [];
  for (let i = 0; i < 32; i++) {
    if (relays[i] === 1) onList.push(`R${i + 1}`);
  }
  return `开启 [${onList.length}/32]: ${onList.length > 0 ? onList.join(', ') : '全部关闭'}`;
}

// 生成通道1的随机温度数据 (例如 18.0℃ ~ 36.5℃，0.1精度)
function getRandomTemperature() {
  const temp = 18.0 + Math.random() * 18.5;
  const tempFixed = parseFloat(temp.toFixed(1));
  const rawValue = Math.round(tempFixed * 10); // 对应 0.1 精度下的 Modbus 整数值
  return { temp: tempFixed, rawValue };
}

/**
 * 标准 Modbus CRC16 计算 (与服务端及嵌入式 C 代码一致)
 */
function crc16(buf) {
  let crc = 0xFFFF;
  for (let i = 0; i < buf.length; i++) {
    crc ^= buf[i];
    for (let j = 0; j < 8; j++) {
      if ((crc & 0x0001) !== 0) {
        crc = (crc >> 1) ^ 0xA001;
      } else {
        crc = crc >> 1;
      }
    }
  }
  const low = crc & 0xFF;
  const high = (crc >> 8) & 0xFF;
  return Buffer.from([low, high]);
}

/**
 * 格式化 Buffer 为 Hex 字符串
 */
function toHex(buf) {
  return Array.from(buf).map(b => b.toString(16).padStart(2, '0').toUpperCase()).join(' ');
}

function log(tag, msg) {
  const time = new Date().toISOString().substring(11, 23);
  console.log(`[${time}] [${tag}] ${msg}`);
}

// 初始化继电器
initRelays(RELAYS_INIT_MODE);

console.log('='.repeat(70));
console.log(`🚀 启动模拟设备客户端 (Control V2)`);
console.log(`📌 目标服务端: ${HOST}:${PORT}`);
console.log(`📌 设备 ID 字节: ${toHex(Buffer.from(DEVICE_ID_BYTES))} (对应设备ID: ${DEVICE_ID_NUM})`);
console.log(`📌 通道数据模式: ${MODE} 通道 (${MODE === '16' ? '0x40 报文' : MODE === '64' ? '0xA1 报文' : '0xA0 报文'})`);
console.log(`📌 通道 1 设定: 动态随机温度 (18.0℃ ~ 36.5℃，0.1精度)`);
console.log(`📌 32个继电器初始化: ${formatRelaysSummary()}`);
console.log(`💡 提示: 运行中可在控制台输入 1~32 手动翻转继电器，输入 r 随机重置，输入 0 全关，输入 1 全开`);
console.log('='.repeat(70));

let client = null;
let recvBuffer = Buffer.alloc(0);
let isExiting = false;

function connect() {
  if (isExiting) return;

  client = new net.Socket();
  recvBuffer = Buffer.alloc(0);

  log('TCP', `正在连接服务端 ${HOST}:${PORT} ...`);

  client.connect(PORT, HOST, () => {
    log('TCP', `✅ 已成功连接到服务端 ${HOST}:${PORT}`);

    if (ACTIVE) {
      log('INFO', `[主动模式] 正在向服务端主动发送设备 ID 报文...`);
      sendDeviceIDResponse();
    } else {
      log('INFO', `[被动模式] 等待服务端下发读取设备 ID 指令 (00 03 00 60 00 04 45 C6)...`);
    }
  });

  client.on('data', (chunk) => {
    recvBuffer = Buffer.concat([recvBuffer, chunk]);
    log('RECV', `📥 收到下行数据 (${chunk.length} 字节): ${toHex(chunk)}`);
    processIncomingData();
  });

  client.on('error', (err) => {
    log('ERROR', `❌ Socket 异常: ${err.message}`);
  });

  client.on('close', () => {
    log('TCP', `⚠️ 与服务端的连接已断开`);
    if (!isExiting && !ONCE) {
      log('TCP', `3 秒后将尝试重新连接...`);
      setTimeout(connect, 3000);
    }
  });
}

function processIncomingData() {
  while (recvBuffer.length >= 4) {
    // 1. 匹配 读取设备 ID 请求: 00 03 00 60 00 04 [CRC_L] [CRC_H] (共 8 字节)
    if (recvBuffer.length >= 8 &&
        recvBuffer[1] === 0x03 && recvBuffer[2] === 0x00 && recvBuffer[3] === 0x60 &&
        recvBuffer[4] === 0x00 && recvBuffer[5] === 0x04) {
      const packet = recvBuffer.subarray(0, 8);
      recvBuffer = recvBuffer.subarray(8);

      log('DISPATCH', `🎯 匹配到 [读取设备ID] 请求: ${toHex(packet)}`);
      sendDeviceIDResponse();
      continue;
    }

    // 2. 匹配 数据轮询请求: 00 03 00 00 [CRC_L] [CRC_H] (共 6 字节)
    if (recvBuffer.length >= 6 &&
        recvBuffer[1] === 0x03 && recvBuffer[2] === 0x00 && recvBuffer[3] === 0x00) {
      const packet = recvBuffer.subarray(0, 6);
      recvBuffer = recvBuffer.subarray(6);

      log('DISPATCH', `🎯 匹配到 [数据轮询] 请求: ${toHex(packet)}`);
      sendPollingDataResponse();
      continue;
    }

    // 3. 匹配 继电器控制指令: 00 10 00 7A ... (共 11 字节)
    if (recvBuffer.length >= 11 &&
        recvBuffer[1] === 0x10 && recvBuffer[2] === 0x00 && recvBuffer[3] === 0x7A) {
      const packet = recvBuffer.subarray(0, 11);
      recvBuffer = recvBuffer.subarray(11);

      const relayIndex = packet[4]; // 0 ~ 31
      const state = packet[8];      // 0 或 1
      
      // 更新本地 32 个继电器中对应路的值
      if (relayIndex >= 0 && relayIndex < 32) {
        relays[relayIndex] = state === 1 ? 1 : 0;
      }

      log('CONTROL', `⚙️  收到继电器控制: R${relayIndex + 1} -> ${state === 1 ? '开启(1)' : '关闭(0)'} | 当前状态: ${formatRelaysSummary()}`);
      sendRelayControlResponse(relayIndex, state);
      continue;
    }

    // 未能匹配已知包头，跳过1字节滑动寻找
    break;
  }
}

/**
 * 响应设备 ID 报文:
 * addr 03 00 60 00 04 ID1 ID2 ID3 ID4 CRC_L CRC_H (共 12 字节)
 */
function sendDeviceIDResponse() {
  const header = Buffer.from([0x00, 0x03, 0x00, 0x60, 0x00, 0x04]);
  const idBytes = Buffer.from(DEVICE_ID_BYTES);
  const body = Buffer.concat([header, idBytes]);
  const crc = crc16(body);
  const response = Buffer.concat([body, crc]);

  log('SEND', `📤 发送 [设备ID响应] (12字节): ${toHex(response)}`);
  log('INFO', `   ➜ 对应解析结果: frame[6..9]=[${toHex(idBytes)}] => 设备ID=${DEVICE_ID_NUM}`);
  client.write(response);
}

/**
 * 响应轮询数据报文
 * - 第一个通道为随机温度数据
 * - 填充 32 个继电器的当前值
 */
function sendPollingDataResponse() {
  const { temp, rawValue } = getRandomTemperature();
  let response;

  if (MODE === '64') {
    // A1 模式: 64 通道 (共 166 字节: 头4 + 数据160 + CRC2)
    // 数据段 160 字节: 128 字节通道 (64通道*2B) + 32 字节继电器
    const header = Buffer.from([0x00, 0x03, 0x00, 0xA1]);
    const data = Buffer.alloc(160);

    // 通道 1 填入随机温度数据 (大端序 2 字节)
    data.writeInt16BE(rawValue, 0);

    // 其余通道 2~64 填入模拟值
    for (let ch = 1; ch < 64; ch++) {
      data.writeInt16BE(100 + ch, ch * 2);
    }

    // 填充 32 个继电器状态 (偏移 128~159)
    for (let i = 0; i < 32; i++) {
      data[128 + i] = relays[i];
    }

    const body = Buffer.concat([header, data]);
    const crc = crc16(body);
    response = Buffer.concat([body, crc]);
    log('SEND', `📤 发送 [64通道 A1 轮询响应] (166字节)`);

  } else if (MODE === '32') {
    // A0 模式: 32 通道 (共 166 字节: 头4 + 数据160 + CRC2)
    // 数据段 160 字节: 128 字节通道 (32通道*4B) + 32 字节继电器
    const header = Buffer.from([0x00, 0x03, 0x00, 0xA0]);
    const data = Buffer.alloc(160);

    // 通道 1 填入随机温度数据 (每通道4字节，前2字节为0，后2字节为数据)
    data.writeInt32BE(rawValue, 0);

    // 其余通道 2~32 填入模拟值
    for (let ch = 1; ch < 32; ch++) {
      data.writeInt32BE(100 + ch, ch * 4);
    }

    // 填充 32 个继电器状态 (偏移 128~159)
    for (let i = 0; i < 32; i++) {
      data[128 + i] = relays[i];
    }

    const body = Buffer.concat([header, data]);
    const crc = crc16(body);
    response = Buffer.concat([body, crc]);
    log('SEND', `📤 发送 [32通道 A0 轮询响应] (166字节)`);

  } else {
    // 16 通道 0x40 模式 (共 70 字节: 头4 + 数据64 + CRC2)
    // 数据段 64 字节: 32 字节通道 (16通道*2B) + 32 字节继电器
    const header = Buffer.from([0x00, 0x03, 0x00, 0x40]);
    const data = Buffer.alloc(64);

    // 通道 1 填入随机温度数据 (大端序 2 字节)
    data.writeInt16BE(rawValue, 0);

    // 其余通道 2~16 填入模拟值
    for (let ch = 1; ch < 16; ch++) {
      data.writeInt16BE(100 + ch, ch * 2);
    }

    // 填充 32 个继电器状态 (偏移 32~63)
    for (let i = 0; i < 32; i++) {
      data[32 + i] = relays[i];
    }

    const body = Buffer.concat([header, data]);
    const crc = crc16(body);
    response = Buffer.concat([body, crc]);
    log('SEND', `📤 发送 [16通道 0x40 轮询响应] (70字节)`);
  }

  log('DATA', `🌡️  通道1(温度): ${temp} ℃ (Modbus原始值: ${rawValue} / 0x${rawValue.toString(16).padStart(4, '0').toUpperCase()})`);
  log('DATA', `🔌 32路继电器状态: ${formatRelaysSummary()}`);
  log('DEBUG', `   ➜ 数据帧前 20 字节: ${toHex(response.subarray(0, 20))} ...`);
  client.write(response);

  if (ONCE) {
    log('INFO', '单次交互测试完成，将在 2 秒后正常退出...');
    setTimeout(() => {
      isExiting = true;
      client.end();
      process.exit(0);
    }, 2000);
  }
}

/**
 * 响应继电器控制:
 * addr 10 00 7A num state CRC_L CRC_H (共 8 字节)
 */
function sendRelayControlResponse(relayIndex, state) {
  const body = Buffer.from([0x00, 0x10, 0x00, 0x7A, relayIndex, state]);
  const crc = crc16(body);
  const response = Buffer.concat([body, crc]);

  log('SEND', `📤 发送 [继电器控制确认] (8字节): ${toHex(response)}`);
  client.write(response);
}

// 支持控制台按键交互手动修改继电器
if (process.stdin.isTTY) {
  process.stdin.setEncoding('utf-8');
  process.stdin.on('data', (d) => {
    const input = d.toString().trim().toLowerCase();
    if (input === 'r') {
      for (let i = 0; i < 32; i++) relays[i] = Math.random() > 0.5 ? 1 : 0;
      log('MANUAL', `🎲 已随机重置 32 个继电器状态: ${formatRelaysSummary()}`);
    } else if (input === '0' || input === 'off') {
      relays.fill(0);
      log('MANUAL', `🔌 已关闭全部 32 个继电器`);
    } else if (input === '1' || input === 'on') {
      relays.fill(1);
      log('MANUAL', `⚡ 已开启全部 32 个继电器`);
    } else {
      const idx = parseInt(input, 10);
      if (!isNaN(idx) && idx >= 1 && idx <= 32) {
        relays[idx - 1] = relays[idx - 1] === 1 ? 0 : 1;
        log('MANUAL', `🔘 手动切换 R${idx} -> ${relays[idx - 1] === 1 ? '开启(1)' : '关闭(0)'} | ${formatRelaysSummary()}`);
      }
    }
  });
}

if (TIMEOUT > 0) {
  setTimeout(() => {
    log('INFO', `测试达到最大限制时间 (${TIMEOUT} 秒)，主动断开退出`);
    isExiting = true;
    client.end();
    setTimeout(() => process.exit(0), 500);
  }, TIMEOUT * 1000);
}

// 优雅退出
process.on('SIGINT', () => {
  console.log('\n用户中断，正在断开连接退出...');
  isExiting = true;
  if (client) client.end();
  process.exit(0);
});

// 启动连接
connect();
