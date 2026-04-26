import {useState, useEffect} from 'react';
import './App.css';
import {ValidateAndSaveConfig, GetConfig, Clone, Status, Commit, Pull, Push, Log, GetRepos, SaveRepos, ListGithubRepos} from '../wailsjs/go/main/App';

interface RepoInfo {
  name: string;
  local_path: string;
  remote_url: string;
  last_sync_time: string;
  branch: string;
  commit_sha: string;
}

interface FileStatus {
  path: string;
  status: string;
  staged: boolean;
  modified: boolean;
}

interface LogEntry {
  sha: string;
  message: string;
  author: string;
  timestamp: string;
}

function App() {
  const [token, setToken] = useState('');
  const [storagePath, setStoragePath] = useState('');
  const [username, setUsername] = useState('');
  const [isConfigured, setIsConfigured] = useState(false);

  const [repos, setRepos] = useState<RepoInfo[]>([]);
  const [selectedRepo, setSelectedRepo] = useState<RepoInfo | null>(null);
  const [cloneUrl, setCloneUrl] = useState('');
  const [cloneName, setCloneName] = useState('');

  const [status, setStatus] = useState<FileStatus[]>([]);
  const [isClean, setIsClean] = useState(true);
  const [commitMsg, setCommitMsg] = useState('');
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    loadConfig();
  }, []);

  async function loadConfig() {
    try {
      const cfg = await GetConfig();
      if (cfg && cfg.token) {
        setToken(cfg.token);
        setUsername(cfg.username || '');
        setStoragePath(cfg.storage_path || '');
        setIsConfigured(true);
        await loadRepos();
      }
    } catch (e) {
      console.error(e);
    }
  }

  async function loadRepos() {
    try {
      const r = await GetRepos();
      setRepos(r || []);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleSaveConfig() {
    if (!token) {
      setError('请输入 GitHub Token');
      return;
    }
    try {
      const user = await ValidateAndSaveConfig(token, storagePath);
      setUsername(user);
      setIsConfigured(true);
      setError('');
      setMessage('配置保存成功！');
    } catch (e: any) {
      const errMsg = e?.message || String(e);
      console.error('Save config error:', errMsg);

      let displayMsg = '保存配置失败';
      if (errMsg.includes('step1_token_validation_failed')) {
        displayMsg = 'Token 验证失败：请检查 Token 是否正确或已过期';
      } else if (errMsg.includes('step2_create_storage_dir_failed')) {
        displayMsg = '创建仓库目录失败：' + errMsg;
      } else if (errMsg.includes('step4_encrypt_failed')) {
        displayMsg = '加密失败：' + errMsg;
      } else if (errMsg.includes('step5_create_config_dir_failed')) {
        displayMsg = '创建配置目录失败：' + errMsg;
      } else if (errMsg.includes('step6_write_config_failed')) {
        displayMsg = '写入配置文件失败：' + errMsg;
      } else {
        displayMsg = '保存配置失败：' + errMsg;
      }
      setError(displayMsg);
    }
  }

  async function handleClone() {
    if (!cloneUrl) {
      setError('请输入仓库地址');
      return;
    }
    const name = cloneName || cloneUrl.split('/').pop()?.replace('.git', '') || 'repo';
    try {
      setMessage('正在克隆仓库...');
      setError('');
      const result = await Clone(cloneUrl, name);
      if (result.success) {
        const newRepo: RepoInfo = {
          name: name,
          local_path: result.local_path,
          remote_url: result.remote_url,
          last_sync_time: new Date().toISOString(),
          branch: 'main',
          commit_sha: '',
        };
        const updatedRepos = [...repos, newRepo];
        setRepos(updatedRepos);
        await SaveRepos(updatedRepos);
        setCloneUrl('');
        setCloneName('');
        setMessage('仓库克隆成功！');
        setError('');
      } else {
        setError('克隆失败：' + (result.error || '未知错误'));
      }
    } catch (e: any) {
      setError('克隆失败：' + (e.message || String(e)));
    }
  }

  async function handleSelectRepo(repo: RepoInfo) {
    setSelectedRepo(repo);
    await refreshStatus(repo.local_path);
    await refreshLog(repo.local_path);
  }

  async function refreshStatus(repoPath: string) {
    try {
      const s = await Status(repoPath);
      setStatus(s.files || []);
      setIsClean(s.clean);
    } catch (e) {
      console.error(e);
    }
  }

  async function refreshLog(repoPath: string) {
    try {
      const l = await Log(repoPath, 50);
      setLogs(l || []);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleCommit() {
    if (!selectedRepo || !commitMsg) return;
    try {
      await Commit(selectedRepo.local_path, commitMsg);
      setCommitMsg('');
      await refreshStatus(selectedRepo.local_path);
      await refreshLog(selectedRepo.local_path);
      setMessage('提交成功！');
    } catch (e: any) {
      setError('提交失败：' + (e.message || String(e)));
    }
  }

  async function handlePull() {
    if (!selectedRepo) return;
    try {
      setMessage('正在拉取更新...');
      await Pull(selectedRepo.local_path);
      await refreshStatus(selectedRepo.local_path);
      await refreshLog(selectedRepo.local_path);
      setMessage('拉取成功！');
    } catch (e: any) {
      setError('拉取失败：' + (e.message || String(e)));
    }
  }

  async function handlePush() {
    if (!selectedRepo) return;
    try {
      setMessage('正在推送...');
      await Push(selectedRepo.local_path);
      setMessage('推送成功！');
    } catch (e: any) {
      setError('推送失败：' + (e.message || String(e)));
    }
  }

  if (!isConfigured) {
    return (
      <div className="app">
        <div className="config-panel">
          <h1>GitHub 同步工具</h1>
          <p>请输入 GitHub Personal Access Token 开始使用</p>
          <a href="https://github.com/settings/tokens" target="_blank" rel="noopener noreferrer">
            在 GitHub 上生成 Token
          </a>
          <div className="form-group">
            <label>GitHub Token</label>
            <input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="ghp_xxxxxxxxxxxx"
            />
          </div>
          <div className="form-group">
            <label>仓库存储路径（可选）</label>
            <input
              type="text"
              value={storagePath}
              onChange={(e) => setStoragePath(e.target.value)}
              placeholder="留空使用默认路径"
            />
          </div>
          <button className="btn-primary" onClick={handleSaveConfig}>
            保存并连接
          </button>
          {error && <div className="error">{error}</div>}
        </div>
      </div>
    );
  }

  return (
    <div className="app">
      <div className="header">
        <h1>GitHub 同步工具</h1>
        <span>登录用户: {username}</span>
      </div>

      {message && <div className="success-message">{message}</div>}
      {error && <div className="error">{error}</div>}

      <div className="main-content">
        <div className="left-panel">
          <div className="clone-section">
            <h3>克隆仓库</h3>
            <input
              type="text"
              value={cloneUrl}
              onChange={(e) => setCloneUrl(e.target.value)}
              placeholder="https://github.com/user/repo"
            />
            <input
              type="text"
              value={cloneName}
              onChange={(e) => setCloneName(e.target.value)}
              placeholder="本地文件夹名称（可选）"
            />
            <button className="btn-primary" onClick={handleClone}>克隆</button>
          </div>

          <div className="repos-section">
            <h3>仓库列表</h3>
            <div className="repo-list">
              {repos.map((repo) => (
                <div
                  key={repo.name}
                  className={`repo-item ${selectedRepo?.name === repo.name ? 'selected' : ''}`}
                  onClick={() => handleSelectRepo(repo)}
                >
                  <span className="repo-name">{repo.name}</span>
                  <span className="repo-path">{repo.local_path}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="right-panel">
          {selectedRepo ? (
            <>
              <div className="repo-info">
                <h3>{selectedRepo.name}</h3>
                <p>{selectedRepo.local_path}</p>
              </div>

              <div className="actions">
                <button onClick={handlePull}>拉取</button>
                <button onClick={handlePush}>推送</button>
              </div>

              <div className="status-section">
                <h4>文件状态 {isClean ? '（已同步）' : `（${status.length} 个文件有更改）`}</h4>
                <div className="file-list">
                  {status.map((f) => (
                    <div key={f.path} className={`file-item ${f.status}`}>
                      <span className="file-status">[{f.status === 'modified' ? '已修改' : f.status === 'staged' ? '已暂存' : f.status === 'untracked' ? '未跟踪' : '未知'}]</span>
                      <span className="file-path">{f.path}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="commit-section">
                <h4>提交更改</h4>
                <textarea
                  value={commitMsg}
                  onChange={(e) => setCommitMsg(e.target.value)}
                  placeholder="提交说明..."
                />
                <button
                  className="btn-primary"
                  onClick={handleCommit}
                  disabled={!commitMsg || isClean}
                >
                  提交
                </button>
              </div>

              <div className="log-section">
                <h4>最近提交记录</h4>
                <div className="log-list">
                  {logs.map((log) => (
                    <div key={log.sha} className="log-item">
                      <span className="log-sha">{log.sha.substring(0, 7)}</span>
                      <span className="log-msg">{log.message}</span>
                      <span className="log-author">{log.author}</span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          ) : (
            <div className="no-repo">
              <p>请选择一个仓库或克隆新仓库</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
