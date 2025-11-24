// 设置上下文菜单项
chrome.runtime.onInstalled.addListener(() => {
    chrome.contextMenus.create({
        id: 'play', // 上下文菜单项的唯一标识符，一块身份牌
        title: '立即播放', // %s 代表选中的那块肉——用户选中的文本
        type: "normal",
        contexts: ['link']
    });
});

chrome.contextMenus.onClicked.addListener((item, tab) => {
    if (item.menuItemId === 'play') {
        console.log('点击了立即播放菜单项，链接地址为:', item.linkUrl);
        chrome.storage.sync.get(['serverUrl'], async (result) => {
            console.log('读取到的服务端地址:', result.serverUrl);
            if (result.serverUrl) {
                console.log('准备打开的URL:', item.linkUrl);
                chrome.tabs.create({
                    url: result.serverUrl + `?link=${encodeURIComponent(item.linkUrl)}`,
                    active: true // true: 立即切换到新标签页, false: 后台打开
                });
            }
        });
    }
});