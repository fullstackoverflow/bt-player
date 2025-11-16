// 加载保存的设置
function loadSettings() {
    chrome.storage.sync.get(['serverUrl'], (result) => {
        if (result.serverUrl) {
            document.getElementById('serverUrl').value = result.serverUrl;
        }
    });
}

// 保存设置
function saveSettings() {
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
    
    // 保存到 chrome.storage
    chrome.storage.sync.set({
        serverUrl: serverUrl
    }, () => {
        showStatus('设置已保存！', true);
    });
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

// 保存按钮点击事件
document.getElementById('save').addEventListener('click', saveSettings);

// 回车键保存
document.getElementById('serverUrl').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        saveSettings();
    }
});
