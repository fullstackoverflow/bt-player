// 加载保存的设置
function loadSettings() {
    chrome.storage.sync.get(['serverUrl', 'isConnected'], (result) => {
        if (result.serverUrl) {
            document.getElementById('serverUrl').value = result.serverUrl;
        }
        
        // 如果之前已连接过，直接尝试加载配置进入第二步
        if (result.isConnected && result.serverUrl) {
            loadConfigAndShowStep2(result.serverUrl);
        }
    });
}

// 加载配置并显示第二步
async function loadConfigAndShowStep2(serverUrl) {
    try {
        const response = await fetch(`${serverUrl}/config`);
        
        if (!response.ok) {
            // 如果连接失败，清除连接状态，保持在第一步
            chrome.storage.sync.set({ isConnected: false });
            return;
        }
        
        const config = await response.json();
        
        // 填充配置数据
        document.getElementById('enableSeeding').checked = config.enable_seeding;
        document.getElementById('storageLimit').value = config.storage_limit || 0;
        
        // 切换到第二步
        document.getElementById('step1').style.display = 'none';
        document.getElementById('step2').style.display = 'block';
        
    } catch (error) {
        // 连接失败，清除连接状态
        chrome.storage.sync.set({ isConnected: false });
    }
}

// 连接服务器并加载配置
async function connectServer() {
    const serverUrl = document.getElementById('serverUrl').value.trim();
    
    // 验证URL格式
    if (!serverUrl) {
        showStatus('请输入服务端地址', false);
        return;
    }
    
    // 简单的URL验证
    try {
        new URL(serverUrl);
    } catch (e) {
        showStatus('请输入有效的URL地址', false);
        return;
    }
    
    // 显示连接中状态
    showStatus('连接中...', true);
    
    try {
        // 请求配置数据
        const response = await fetch(`${serverUrl}/config`);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const config = await response.json();
        
        // 保存服务器地址和连接状态到 chrome.storage
        chrome.storage.sync.set({ 
            serverUrl: serverUrl,
            isConnected: true 
        });
        
        // 填充配置数据
        document.getElementById('enableSeeding').checked = config.enable_seeding;
        document.getElementById('storageLimit').value = config.storage_limit || 0;
        
        // 切换到第二步
        document.getElementById('step1').style.display = 'none';
        document.getElementById('step2').style.display = 'block';
        
    } catch (error) {
        showStatus(`连接失败: ${error.message}`, false);
    }
}

// 返回第一步
function backToStep1() {
    // 清除连接状态
    chrome.storage.sync.set({ isConnected: false });
    
    document.getElementById('step2').style.display = 'none';
    document.getElementById('step1').style.display = 'block';
}

// 显示状态消息
function showStatus(message, isSuccess) {
    const statusDiv = document.getElementById('status');
    statusDiv.textContent = message;
    statusDiv.className = 'status';
    statusDiv.style.display = 'block';
    
    if (isSuccess) {
        statusDiv.classList.add('success');
    } else {
        statusDiv.classList.add('error');
    }
    
    // 2秒后隐藏
    setTimeout(() => {
        statusDiv.style.display = 'none';
    }, 2000);
}

// 页面加载时读取设置
document.addEventListener('DOMContentLoaded', loadSettings);

// 连接按钮点击事件
document.getElementById('connect').addEventListener('click', connectServer);

// 回车键连接
document.getElementById('serverUrl').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        connectServer();
    }
});

// 返回按钮点击事件
document.getElementById('back').addEventListener('click', backToStep1);

// 保存配置按钮点击事件
document.getElementById('saveConfig').addEventListener('click', saveServerConfig);

// 保存配置按钮点击事件
document.getElementById('saveConfig').addEventListener('click', saveServerConfig);
async function saveServerConfig() {
    const serverUrl = document.getElementById('serverUrl').value.trim();
    const enableSeeding = document.getElementById('enableSeeding').checked;
    const storageLimit = parseInt(document.getElementById('storageLimit').value) || 0;
    
    if (storageLimit < 0) {
        showConfigStatus('存储上限不能为负数', false);
        return;
    }
    
    try {
        const response = await fetch(`${serverUrl}/config`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                enable_seeding: enableSeeding,
                storage_limit: storageLimit
            })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const result = await response.json();
        
        showConfigStatus('配置保存成功', true);
    } catch (error) {
        showConfigStatus(`保存失败: ${error.message}`, false);
    }
}

// 显示配置状态消息
function showConfigStatus(message, isSuccess) {
    const statusDiv = document.getElementById('configStatus');
    statusDiv.textContent = message;
    statusDiv.className = 'status';
    statusDiv.style.display = 'block';
    
    if (isSuccess) {
        statusDiv.classList.add('success');
    } else {
        statusDiv.classList.add('error');
    }
    
    setTimeout(() => {
        statusDiv.style.display = 'none';
    }, 3000);
}
