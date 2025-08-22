document.addEventListener('DOMContentLoaded', () => {
    const searchButton = document.getElementById('searchButton');
    const wordInput = document.getElementById('wordInput');
    const resultsDiv = document.getElementById('results');
    const sourceLangSelect = document.getElementById('sourceLang');
    const targetLangSelect = document.getElementById('targetLang');
    
    // 全局配置变量，增加 available_pairs
    let appConfig = {
        max_input_chars: 50,
        available_pairs: {}
    };

    // --- 新增：一个映射，用于在UI上显示完整的语言名称 ---
    const langNameMap = {
        "en": "英语 (English)", "zh": "中文 (Chinese)", "es": "西班牙语 (Spanish)",
        "fr": "法语 (French)", "de": "德语 (German)", "ru": "俄语 (Russian)",
        "ja": "日语 (Japanese)", "ar": "阿拉伯语 (Arabic)", "pt": "葡萄牙语 (Portuguese)",
    };
    // --- 映射结束 ---

    // --- 新增：更新目标语言下拉菜单的函数 ---
    function updateTargetLangOptions() {
        const selectedSource = sourceLangSelect.value;
        const availableTargets = appConfig.available_pairs[selectedSource] || [];

        // 清空当前选项
        targetLangSelect.innerHTML = '';

        if (availableTargets.length === 0) {
            // 如果没有可用的目标语言，可以禁用或显示提示
            const option = document.createElement('option');
            option.textContent = "无可用目标";
            targetLangSelect.appendChild(option);
            targetLangSelect.disabled = true;
        } else {
            // 重新填充选项
            availableTargets.forEach(langCode => {
                const option = document.createElement('option');
                option.value = langCode;
                option.textContent = langNameMap[langCode] || langCode;
                targetLangSelect.appendChild(option);
            });
            targetLangSelect.disabled = false;
        }
    }
    // --- 函数结束 ---


    (async function fetchConfig() {
        try {
            const response = await fetch('/api/config');
            if (response.ok) {
                const configData = await response.json();
                appConfig = configData;
                console.log("Successfully loaded config from server:", appConfig);
                
                // --- 新增：配置加载成功后，立即更新一次目标语言菜单 ---
                updateTargetLangOptions();
            }
        } catch (error) {
            console.error("Could not fetch config from server, using default values.", error);
        }
    })();
    
    // --- 新增：为源语言选择框添加 onchange 事件监听器 ---
    sourceLangSelect.addEventListener('change', updateTargetLangOptions);


    const performSearch = async () => {
        // ... performSearch 函数本身的代码完全不用变 ...
        const word = wordInput.value.trim();
        const sourceLang = sourceLangSelect.value;
        const targetLang = targetLangSelect.value;
        if (!word || !sourceLang || !targetLang || targetLangSelect.disabled) return;

        const maxChars = appConfig.max_input_chars;
        if (word.length > maxChars) {
            resultsDiv.innerHTML = `<p style="color: red;">输入内容过长，最多允许输入 ${maxChars} 个字符。</p>`;
            return;
        }

        resultsDiv.innerHTML = '<p>正在查询...</p>';
        try {
            const apiUrl = `/api/lookup?word=${encodeURIComponent(word)}&source=${sourceLang}&target=${targetLang}`;
            const response = await fetch(apiUrl);
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`服务器错误: ${response.status} - ${errorText}`);
            }
            const data = await response.json();
            console.log("Received data from backend:", data); 
            renderResults(data);
        } catch (error) {
            resultsDiv.innerHTML = `<p style="color: red;">查询失败: ${error.message}</p>`;
        }
    };

    // ... 其他事件监听和 renderResults 函数保持不变 ...
    searchButton.addEventListener('click', performSearch);
    wordInput.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
            performSearch();
        }
    });

    function renderResults(data) {
        const resultsDiv = document.getElementById('results');
        resultsDiv.innerHTML = ''; 
        let html = `<h2>${wordInput.value}</h2>`;
        html += `<div class="phonetics">`;
        if (data.p) {
            html += `<span>${data.p}</span>`;
        }
        html += `</div>`;
        if (data.defs && Array.isArray(data.defs) && data.defs.length > 0) {
            data.defs.forEach(def => {
                html += `<div class="entry">`;
                html += `<div class="part-of-speech">${def.pos}</div>`;
                html += `<div class="definition-block">`;
                html += `<p>• ${def.m}</p>`;
                if (def.ex) {
                    html += `<div class="example"><p>e.g., ${def.ex}</p></div>`;
                }
                html += `</div>`;
                html += `</div>`;
            });
        } else {
             html += `<p>未找到该词的释义。</p>`
        }
        resultsDiv.innerHTML = html;
    }
});