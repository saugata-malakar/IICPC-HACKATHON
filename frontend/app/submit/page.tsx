"use client";

import { useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useDropzone } from "react-dropzone";

interface UploadStatus {
  stage: "idle" | "uploading" | "scanning" | "building" | "deploying" | "completed" | "failed";
  progress: number;
  message: string;
}

export default function SubmitPage() {
  const [file, setFile] = useState<File | null>(null);
  const [status, setStatus] = useState<UploadStatus>({
    stage: "idle",
    progress: 0,
    message: "",
  });
  const [submissionId, setSubmissionId] = useState<string>("");

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > 0) {
      setFile(acceptedFiles[0]);
      setStatus({ stage: "idle", progress: 0, message: "" });
    }
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      "application/zip": [".zip"],
      "application/x-tar": [".tar", ".tar.gz", ".tgz"],
      "application/wasm": [".wasm"],
    },
    maxFiles: 1,
    maxSize: 50 * 1024 * 1024, // 50MB
  });

  const handleSubmit = async () => {
    if (!file) return;

    const stages: Array<{ stage: UploadStatus["stage"]; message: string; duration: number }> = [
      { stage: "uploading", message: "Uploading submission...", duration: 2000 },
      { stage: "scanning", message: "Running static security scan...", duration: 3000 },
      { stage: "building", message: "Building Docker container...", duration: 5000 },
      { stage: "deploying", message: "Deploying to sandbox...", duration: 3000 },
      { stage: "completed", message: "Submission deployed successfully!", duration: 0 },
    ];

    for (const { stage, message, duration } of stages) {
      setStatus({ stage, progress: 0, message });

      // Simulate progress
      const steps = 20;
      for (let i = 0; i <= steps; i++) {
        await new Promise((resolve) => setTimeout(resolve, duration / steps));
        setStatus((prev) => ({ ...prev, progress: (i / steps) * 100 }));
      }
    }

    // Generate submission ID
    const id = `sub-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    setSubmissionId(id);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-900 text-white p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <motion.div initial={{ opacity: 0, y: -20 }} animate={{ opacity: 1, y: 0 }} className="mb-12">
          <h1 className="text-5xl font-black bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400 mb-4">
            Submit Your Code
          </h1>
          <p className="text-slate-400 text-lg">
            Upload your trading infrastructure (ZIP, TAR, or WASM). Max 50MB.
          </p>
        </motion.div>

        {/* Upload Zone */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.2 }}
        >
          <div
            {...getRootProps()}
            className={`border-2 border-dashed rounded-2xl p-12 text-center cursor-pointer transition-all ${
              isDragActive
                ? "border-blue-400 bg-blue-500/10"
                : file
                ? "border-green-400 bg-green-500/10"
                : "border-slate-600 bg-slate-900/30 hover:border-slate-500"
            }`}
          >
            <input {...getInputProps()} />
            <motion.div
              animate={{ scale: isDragActive ? 1.1 : 1 }}
              transition={{ type: "spring", stiffness: 300 }}
            >
              {file ? (
                <>
                  <div className="text-6xl mb-4">✅</div>
                  <p className="text-xl font-bold text-green-300 mb-2">{file.name}</p>
                  <p className="text-slate-400">{(file.size / 1024 / 1024).toFixed(2)} MB</p>
                </>
              ) : (
                <>
                  <div className="text-6xl mb-4">📦</div>
                  <p className="text-xl font-bold mb-2">
                    {isDragActive ? "Drop your file here" : "Drag & drop your submission"}
                  </p>
                  <p className="text-slate-400">or click to browse</p>
                  <div className="mt-4 flex items-center justify-center gap-3 text-sm text-slate-500">
                    <span>.zip</span>
                    <span>•</span>
                    <span>.tar.gz</span>
                    <span>•</span>
                    <span>.wasm</span>
                  </div>
                </>
              )}
            </motion.div>
          </div>
        </motion.div>

        {/* Submit Button */}
        {file && status.stage === "idle" && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-8 flex gap-4"
          >
            <button
              onClick={handleSubmit}
              className="flex-1 bg-gradient-to-r from-blue-500 to-purple-500 hover:from-blue-600 hover:to-purple-600 text-white font-bold py-4 px-8 rounded-xl transition-all shadow-lg shadow-blue-500/50 hover:shadow-xl hover:shadow-blue-500/70"
            >
              🚀 Submit to Platform
            </button>
            <button
              onClick={() => {
                setFile(null);
                setStatus({ stage: "idle", progress: 0, message: "" });
              }}
              className="bg-slate-800 hover:bg-slate-700 text-white font-bold py-4 px-8 rounded-xl transition-all"
            >
              Clear
            </button>
          </motion.div>
        )}

        {/* Progress Pipeline */}
        <AnimatePresence>
          {status.stage !== "idle" && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              className="mt-12 bg-slate-900/50 backdrop-blur-xl border border-slate-700/50 rounded-2xl p-8"
            >
              <h2 className="text-2xl font-bold mb-6">Deployment Pipeline</h2>

              {/* Stages */}
              <div className="space-y-4 mb-8">
                {["uploading", "scanning", "building", "deploying", "completed"].map((stage, idx) => {
                  const isActive = status.stage === stage;
                  const isCompleted =
                    ["uploading", "scanning", "building", "deploying", "completed"].indexOf(status.stage) > idx;

                  return (
                    <motion.div
                      key={stage}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: idx * 0.1 }}
                      className={`flex items-center gap-4 p-4 rounded-xl ${
                        isActive
                          ? "bg-blue-500/20 border border-blue-400/30"
                          : isCompleted
                          ? "bg-green-500/20 border border-green-400/30"
                          : "bg-slate-800/30 border border-slate-700/30"
                      }`}
                    >
                      <div
                        className={`w-10 h-10 rounded-full flex items-center justify-center font-bold ${
                          isActive
                            ? "bg-blue-500 text-white animate-pulse"
                            : isCompleted
                            ? "bg-green-500 text-white"
                            : "bg-slate-700 text-slate-400"
                        }`}
                      >
                        {isCompleted ? "✓" : idx + 1}
                      </div>
                      <div className="flex-1">
                        <div className="font-semibold capitalize">{stage.replace("_", " ")}</div>
                        {isActive && <div className="text-sm text-slate-400 mt-1">{status.message}</div>}
                      </div>
                      {isActive && (
                        <div className="text-blue-300 font-mono text-sm">{status.progress.toFixed(0)}%</div>
                      )}
                    </motion.div>
                  );
                })}
              </div>

              {/* Progress Bar */}
              {status.stage !== "completed" && status.stage !== "failed" && (
                <div className="bg-slate-800 rounded-full h-3 overflow-hidden">
                  <motion.div
                    className="h-full bg-gradient-to-r from-blue-500 to-purple-500"
                    initial={{ width: 0 }}
                    animate={{ width: `${status.progress}%` }}
                    transition={{ duration: 0.3 }}
                  />
                </div>
              )}

              {/* Completion */}
              {status.stage === "completed" && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="bg-gradient-to-br from-green-500/20 to-emerald-500/20 border border-green-400/30 rounded-xl p-6 text-center"
                >
                  <div className="text-6xl mb-4">🎉</div>
                  <h3 className="text-2xl font-bold text-green-300 mb-2">Deployment Successful!</h3>
                  <p className="text-slate-400 mb-4">Your submission is now live and ready for benchmarking.</p>
                  <div className="bg-slate-900/50 rounded-lg p-4 font-mono text-sm">
                    <span className="text-slate-500">Submission ID:</span>{" "}
                    <span className="text-green-300">{submissionId}</span>
                  </div>
                  <div className="mt-6 flex gap-4 justify-center">
                    <button className="bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition-all">
                      View on Leaderboard
                    </button>
                    <button className="bg-slate-700 hover:bg-slate-600 text-white font-bold py-3 px-6 rounded-lg transition-all">
                      View Analytics
                    </button>
                  </div>
                </motion.div>
              )}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Requirements */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.4 }}
          className="mt-12 bg-slate-900/30 backdrop-blur-xl border border-slate-700/50 rounded-2xl p-8"
        >
          <h2 className="text-2xl font-bold mb-6">Submission Requirements</h2>
          <div className="grid md:grid-cols-2 gap-6">
            <div>
              <h3 className="font-semibold text-blue-300 mb-3">✅ Supported Formats</h3>
              <ul className="space-y-2 text-slate-400">
                <li>• Docker image (tar.gz)</li>
                <li>• Source code (zip)</li>
                <li>• WebAssembly (wasm)</li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold text-purple-300 mb-3">📋 Requirements</h3>
              <ul className="space-y-2 text-slate-400">
                <li>• Expose port 8888</li>
                <li>• Support REST/WebSocket/FIX</li>
                <li>• Max 50MB file size</li>
              </ul>
            </div>
          </div>
        </motion.div>
      </div>
    </div>
  );
}
