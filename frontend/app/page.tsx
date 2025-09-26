'use client'
import React, { useEffect, useState } from 'react'

const NODES = [
  'http://localhost:8081',
  'http://localhost:8082',
  'http://localhost:8083',
];

export default function Home() {
  const [files, setFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [stats, setStats] = useState<{ totalFiles: number, totalBytes: number, quotaBytes: number } | null>(null);
  const [replicaStatus, setReplicaStatus] = useState<Record<string, boolean>>({});
  const [leaderName, setLeaderName] = useState<string>('Loading...');
  const [leaderURL, setLeaderURL] = useState<string>('');
  const [clocks, setClocks] = useState([
    { node: 'A', clock: 0 },
    { node: 'B', clock: 0 }
  ]);

  useEffect(() => {
    checkReplicas();
    findLeader();
    const interval = setInterval(() => {
      checkReplicas();
      findLeader();
    }, 5000);
    const tick = setInterval(() => {
      setClocks(prev => prev.map(c => ({ ...c, clock: c.clock + 1 })));
    }, 3000);
    return () => {
      clearInterval(interval);
      clearInterval(tick);
    };
  }, []);

  useEffect(() => {
    if (leaderURL) {
      fetchFiles();
      fetchStats();
    }
  }, [leaderURL]);

  const findLeader = async () => {
    for (const node of NODES) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 3000);
        const res = await fetch(node + "/status", { signal: controller.signal });
        clearTimeout(timeoutId);
        if (res.ok) {
          const data = await res.json();
          if (data?.is_leader) {
            setLeaderURL(node);
            setLeaderName(`Leader @ ${node}`);
            return;
          }
        }
      } catch (err) {
        console.warn(`❌ Failed to fetch status from ${node}`);
      }
    }
    setLeaderURL('');
    setLeaderName('Unknown');
  };

  const fetchFiles = async () => {
    if (!leaderURL) return;
    try {
      const res = await fetch(`${leaderURL}/files`);
      const data = await res.json();
      // backup server returns array of objects: { name, size, last_modified }
      const names = Array.isArray(data) ? data.map((f: any) => f.name || f.Name).filter(Boolean) : [];
      setFiles(names);
    } catch (err) {
      console.error('❌ Failed to fetch files', err);
      setFiles([]);
    }
  };

  const fetchStats = async () => {
    if (!leaderURL) return;
    try {
      // stats endpoint not provided by backup server; derive minimal stats
      const res = await fetch(`${leaderURL}/files`);
      const data = await res.json();
      const names = Array.isArray(data) ? data.map((f: any) => f.name || f.Name).filter(Boolean) : [];
      setStats({ totalFiles: names.length, totalBytes: 0, quotaBytes: 1 });
    } catch (err) {
      setStats(null);
    }
  };

  const checkReplicas = async () => {
    const status: Record<string, boolean> = {};
    for (const url of NODES) {
      try {
        const res = await fetch(`${url}/status`);
        status[url] = res.ok;
      } catch {
        status[url] = false;
      }
    }
    setReplicaStatus(status);
  };

  const handleUpload = async () => {
    if (!selectedFile) return alert('Please select a file');
    if (!leaderURL) return alert('No leader available');

    try {
      const res = await fetch(`${leaderURL}/upload/${encodeURIComponent(selectedFile.name)}`, {
        method: 'POST',
        body: selectedFile,
      });
      if (res.ok) {
        alert('✅ File uploaded');
        fetchFiles();
        fetchStats();
      } else {
        alert('❌ Upload failed');
      }
    } catch (err) {
      alert('❌ Upload error');
      console.error(err);
    }
  };

  const handleDownload = async (filename: string) => {
    if (!leaderURL) return;
    
    try {
      const response = await fetch(`${leaderURL}/${encodeURIComponent(filename)}`);
      
      if (!response.ok) {
        alert('❌ Failed to download file');
        return;
      }

      // Get the blob data from response
      const blob = await response.blob();
      
      // Create a temporary URL for the blob
      const url = window.URL.createObjectURL(blob);
      
      // Create a temporary anchor element and trigger download
      const link = document.createElement('a');
      link.href = url;
      link.download = filename; // This sets the filename for download
      document.body.appendChild(link);
      link.click();
      
      // Clean up
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      
    } catch (err) {
      console.error('❌ Download error:', err);
      alert('❌ Failed to download file');
    }
  };

  // Delete endpoint not implemented in backup server.

  const sendMessage = (from: string, to: string) => {
    setClocks(prev => {
      const fromClock = prev.find(c => c.node === from)?.clock ?? 0;
      return prev.map(c => {
        if (c.node === to) {
          return { ...c, clock: Math.max(c.clock, fromClock) + 1 };
        }
        return c;
      });
    });
  };

  return (
    <main style={{
      padding: '3rem',
      fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
      maxWidth: '1200px',
      margin: '0 auto',
      background: '#ffffff',
      minHeight: '100vh',
      color: '#1a1a1a',
      lineHeight: '1.6'
    }}>
      <header style={{ 
        marginBottom: '3rem', 
        textAlign: 'center',
        borderBottom: '2px solid #e5e5e5',
        paddingBottom: '2rem'
      }}>
        <h1 style={{ 
          fontSize: '3rem', 
          fontWeight: '700',
          color: '#000',
          margin: '0',
          letterSpacing: '-0.025em'
        }}>
          📁 Distributed File System
        </h1>
        <p style={{
          fontSize: '1.1rem',
          color: '#666',
          marginTop: '0.5rem',
          fontWeight: '400'
        }}>
          Professional File Management & Replication System
        </p>
      </header>

      {/* Leader Info */}
      <section style={{
        marginBottom: '3rem',
        padding: '1.5rem 2rem',
        background: '#f8f9fa',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        textAlign: 'center',
        boxShadow: '0 2px 4px rgba(0,0,0,0.05)'
      }}>
        <h3 style={{
          margin: '0 0 0.5rem 0',
          fontSize: '1.1rem',
          fontWeight: '600',
          color: '#495057'
        }}>
          Current System Leader
        </h3>
        <div style={{
          fontSize: '1.2rem',
          fontWeight: '700',
          color: '#000'
        }}>
          👑 {leaderName}
        </div>
      </section>

      {/* Upload */}
      <section style={{ 
        marginBottom: '3rem',
        padding: '2rem',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        background: '#ffffff',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
      }}>
        <h2 style={{
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#000',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          📤 Upload File
        </h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
          <input 
            type="file" 
            onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
            style={{
              padding: '0.75rem',
              border: '2px solid #dee2e6',
              borderRadius: '6px',
              fontSize: '0.95rem',
              flex: '1',
              minWidth: '200px',
              backgroundColor: '#ffffff',
              cursor: 'pointer'
            }}
          />
          <button
            onClick={handleUpload}
            style={{
              background: '#000000',
              color: 'white',
              border: 'none',
              padding: '0.75rem 2rem',
              borderRadius: '6px',
              cursor: 'pointer',
              fontWeight: '600',
              fontSize: '0.95rem',
              transition: 'all 0.2s ease',
              boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
            }}
            onMouseOver={(e) => e.currentTarget.style.background = '#333333'}
            onMouseOut={(e) => e.currentTarget.style.background = '#000000'}
          >
            Upload File
          </button>
        </div>
      </section>

      {/* Storage Usage */}
      <section style={{ 
        marginBottom: '3rem',
        padding: '2rem',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        background: '#ffffff',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
      }}>
        <h2 style={{
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#000',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          📦 Storage Usage
        </h2>
        {stats ? (
          <div style={{ display: 'grid', gap: '1rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '1rem' }}>
              <div style={{
                padding: '1rem',
                background: '#f8f9fa',
                borderRadius: '6px',
                textAlign: 'center',
                border: '1px solid #dee2e6'
              }}>
                <div style={{ fontSize: '2rem', fontWeight: '700', color: '#000' }}>{stats.totalFiles}</div>
                <div style={{ fontSize: '0.875rem', color: '#666', fontWeight: '500' }}>Total Files</div>
              </div>
              <div style={{
                padding: '1rem',
                background: '#f8f9fa',
                borderRadius: '6px',
                textAlign: 'center',
                border: '1px solid #dee2e6'
              }}>
                <div style={{ fontSize: '2rem', fontWeight: '700', color: '#000' }}>
                  {(stats.totalBytes / 1024 / 1024).toFixed(1)}
                </div>
                <div style={{ fontSize: '0.875rem', color: '#666', fontWeight: '500' }}>MB Used</div>
              </div>
              <div style={{
                padding: '1rem',
                background: '#f8f9fa',
                borderRadius: '6px',
                textAlign: 'center',
                border: '1px solid #dee2e6'
              }}>
                <div style={{ fontSize: '2rem', fontWeight: '700', color: '#000' }}>
                  {((stats.totalBytes / stats.quotaBytes) * 100).toFixed(1)}%
                </div>
                <div style={{ fontSize: '0.875rem', color: '#666', fontWeight: '500' }}>Quota Used</div>
              </div>
            </div>
            <div>
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '0.5rem'
              }}>
                <span style={{ fontSize: '0.875rem', color: '#666' }}>Storage Progress</span>
                <span style={{ fontSize: '0.875rem', color: '#666' }}>
                  {(stats.totalBytes / 1024 / 1024).toFixed(2)} MB / {(stats.quotaBytes / 1024 / 1024)} MB
                </span>
              </div>
              <div style={{
                background: '#e9ecef',
                height: '12px',
                borderRadius: '6px',
                overflow: 'hidden'
              }}>
                <div style={{
                  width: `${Math.min((stats.totalBytes / stats.quotaBytes) * 100, 100)}%`,
                  height: '12px',
                  background: stats.totalBytes / stats.quotaBytes > 0.8 ? '#dc3545' : '#000000',
                  borderRadius: '6px',
                  transition: 'all 0.3s ease'
                }} />
              </div>
            </div>
          </div>
        ) : (
          <div style={{ 
            textAlign: 'center', 
            padding: '2rem',
            color: '#666',
            fontSize: '1rem'
          }}>
            Loading storage statistics...
          </div>
        )}
      </section>

      {/* Replica Status */}
      <section style={{ 
        marginBottom: '3rem',
        padding: '2rem',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        background: '#ffffff',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
      }}>
        <h2 style={{
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#000',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          🔁 Replica Status
        </h2>
        <div style={{ display: 'grid', gap: '0.75rem' }}>
          {Object.entries(replicaStatus).map(([url, alive]) => (
            <div key={url} style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '1rem 1.25rem',
              background: alive ? '#f8f9fa' : '#fff5f5',
              border: `1px solid ${alive ? '#dee2e6' : '#fecaca'}`,
              borderRadius: '6px',
              transition: 'all 0.2s ease'
            }}>
              <span style={{
                fontSize: '1rem',
                fontWeight: '500',
                color: '#000'
              }}>
                {url.replace('http://localhost:', 'Node ')}
              </span>
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.5rem'
              }}>
                <div style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  background: alive ? '#28a745' : '#dc3545'
                }} />
                <span style={{
                  fontSize: '0.875rem',
                  fontWeight: '600',
                  color: alive ? '#28a745' : '#dc3545'
                }}>
                  {alive ? 'ONLINE' : 'OFFLINE'}
                </span>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* File List */}
      <section style={{ 
        marginBottom: '3rem',
        padding: '2rem',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        background: '#ffffff',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
      }}>
        <h2 style={{
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#000',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          📄 File Management
        </h2>
        {files.length === 0 ? (
          <div style={{ 
            textAlign: 'center', 
            padding: '3rem 1rem',
            color: '#666'
          }}>
            <div style={{ 
              fontSize: '3rem', 
              marginBottom: '1rem',
              opacity: '0.3' 
            }}>📁</div>
            <p style={{ 
              fontSize: '1.1rem',
              fontWeight: '500',
              margin: '0'
            }}>
              No files uploaded yet
            </p>
            <p style={{ 
              fontSize: '0.9rem',
              color: '#999',
              margin: '0.5rem 0 0 0'
            }}>
              Upload your first file to get started
            </p>
          </div>
        ) : (
          <div style={{ 
            border: '1px solid #dee2e6',
            borderRadius: '6px',
            overflow: 'hidden'
          }}>
            {files.map((file, index) => (
              <div key={file} style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '1rem 1.25rem',
                borderBottom: index < files.length - 1 ? '1px solid #f1f3f4' : 'none',
                background: '#ffffff',
                transition: 'background-color 0.2s ease'
              }}
              onMouseOver={(e) => e.currentTarget.style.backgroundColor = '#f8f9fa'}
              onMouseOut={(e) => e.currentTarget.style.backgroundColor = '#ffffff'}
              >
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.75rem'
                }}>
                  <div style={{
                    width: '32px',
                    height: '32px',
                    background: '#f8f9fa',
                    borderRadius: '6px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '1rem'
                  }}>
                    📄
                  </div>
                  <span style={{
                    fontSize: '1rem',
                    fontWeight: '500',
                    color: '#000'
                  }}>
                    {file}
                  </span>
                </div>
                <div style={{ 
                  display: 'flex', 
                  gap: '0.5rem',
                  alignItems: 'center'
                }}>
                  <button 
                    onClick={() => handleDownload(file)} 
                    style={{
                      background: '#ffffff',
                      color: '#000',
                      padding: '0.5rem 1rem',
                      borderRadius: '6px',
                      border: '1px solid #dee2e6',
                      cursor: 'pointer',
                      fontSize: '0.875rem',
                      fontWeight: '500',
                      transition: 'all 0.2s ease'
                    }}
                    onMouseOver={(e) => {
                      e.currentTarget.style.background = '#000000';
                      e.currentTarget.style.color = '#ffffff';
                    }}
                    onMouseOut={(e) => {
                      e.currentTarget.style.background = '#ffffff';
                      e.currentTarget.style.color = '#000000';
                    }}
                  >
                    Download
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Lamport Clock */}
      <section style={{
        marginBottom: '2rem',
        padding: '2rem',
        border: '1px solid #dee2e6',
        borderRadius: '8px',
        background: '#ffffff',
        boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
      }}>
        <h2 style={{
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#000',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          🕒 Lamport Clock Simulation
        </h2>
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: '1rem',
          marginBottom: '1.5rem'
        }}>
          {clocks.map(c => (
            <div key={c.node} style={{
              padding: '1.5rem',
              background: '#f8f9fa',
              borderRadius: '8px',
              textAlign: 'center',
              border: '1px solid #dee2e6'
            }}>
              <div style={{
                fontSize: '2.5rem',
                fontWeight: '700',
                color: '#000',
                marginBottom: '0.5rem'
              }}>
                {c.clock}
              </div>
              <div style={{
                fontSize: '1rem',
                fontWeight: '600',
                color: '#495057'
              }}>
                Node {c.node}
              </div>
            </div>
          ))}
        </div>
        <div style={{ 
          display: 'flex', 
          gap: '1rem',
          justifyContent: 'center',
          flexWrap: 'wrap'
        }}>
          <button 
            onClick={() => sendMessage('A', 'B')} 
            style={{
              background: '#000000',
              color: 'white',
              border: 'none',
              padding: '0.75rem 1.5rem',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '0.95rem',
              fontWeight: '600',
              transition: 'all 0.2s ease',
              boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
            }}
            onMouseOver={(e) => e.currentTarget.style.background = '#333333'}
            onMouseOut={(e) => e.currentTarget.style.background = '#000000'}
          >
            Send A → B
          </button>
          <button 
            onClick={() => sendMessage('B', 'A')}
            style={{
              background: '#000000',
              color: 'white',
              border: 'none',
              padding: '0.75rem 1.5rem',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '0.95rem',
              fontWeight: '600',
              transition: 'all 0.2s ease',
              boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
            }}
            onMouseOver={(e) => e.currentTarget.style.background = '#333333'}
            onMouseOut={(e) => e.currentTarget.style.background = '#000000'}
          >
            Send B → A
          </button>
        </div>
        <div style={{
          marginTop: '1.5rem',
          padding: '1rem',
          background: '#f8f9fa',
          borderRadius: '6px',
          fontSize: '0.875rem',
          color: '#666',
          textAlign: 'center'
        }}>
          <strong>How it works:</strong> When a message is sent, the receiving node updates its clock to ensure proper event ordering in the distributed system.
        </div>
      </section>
    </main>
  )
}