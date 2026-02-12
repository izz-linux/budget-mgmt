import { useState, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Upload, FileSpreadsheet, Check, AlertTriangle } from 'lucide-react';
import styles from './ImportWizard.module.css';

interface ParsedBill {
  name: string;
  due_day: number | null;
  is_autopay: boolean;
  default_amount: number | null;
  category: string;
  credit_card?: {
    card_label: string;
    statement_day: number;
    due_day: number;
    issuer: string;
  };
}

interface ImportPreview {
  bills: ParsedBill[];
  period_count: number;
  warnings: string[];
}

export function ImportWizard() {
  const queryClient = useQueryClient();
  const [step, setStep] = useState<'upload' | 'preview' | 'done'>('upload');
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [error, setError] = useState('');
  const [dragOver, setDragOver] = useState(false);

  const uploadMutation = useMutation({
    mutationFn: async (formData: FormData) => {
      const res = await fetch('/api/v1/import/xlsx', { method: 'POST', body: formData });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error?.message || 'Upload failed');
      }
      const json = await res.json();
      return json.data as ImportPreview;
    },
    onSuccess: (data) => {
      setPreview(data);
      setStep('preview');
    },
    onError: (err: Error) => setError(err.message),
  });

  const confirmMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/import/xlsx/confirm', { method: 'POST' });
      if (!res.ok) throw new Error('Confirm failed');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries();
      setStep('done');
    },
    onError: (err: Error) => setError(err.message),
  });

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const f = e.dataTransfer.files[0];
    if (f && (f.name.endsWith('.xlsx') || f.name.endsWith('.xls'))) {
      setFile(f);
      setError('');
    } else {
      setError('Please drop an .xlsx file');
    }
  }, []);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (f) {
      setFile(f);
      setError('');
    }
  };

  const handleUpload = () => {
    if (!file) return;
    const formData = new FormData();
    formData.append('file', file);
    uploadMutation.mutate(formData);
  };

  if (step === 'done') {
    return (
      <div className={styles.container}>
        <div className={styles.doneCard}>
          <div className={styles.doneIcon}><Check size={48} /></div>
          <h2>Import Complete</h2>
          <p>Your spreadsheet data has been imported successfully.</p>
          <button className={styles.primaryBtn} onClick={() => { setStep('upload'); setFile(null); setPreview(null); }}>
            Import Another
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <h1>Import Spreadsheet</h1>
      <p className={styles.subtitle}>
        Import your existing Budget.xlsx to migrate your data into the app.
      </p>

      {step === 'upload' && (
        <>
          <div
            className={`${styles.dropZone} ${dragOver ? styles.dropZoneActive : ''}`}
            onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
          >
            <Upload size={40} className={styles.dropIcon} />
            <p>Drag & drop your .xlsx file here</p>
            <span className={styles.or}>or</span>
            <label className={styles.fileLabel}>
              Browse files
              <input type="file" accept=".xlsx,.xls" onChange={handleFileSelect} hidden />
            </label>
          </div>

          {file && (
            <div className={styles.fileInfo}>
              <FileSpreadsheet size={20} />
              <span>{file.name}</span>
              <span className={styles.fileSize}>{(file.size / 1024).toFixed(0)} KB</span>
              <button className={styles.primaryBtn} onClick={handleUpload} disabled={uploadMutation.isPending}>
                {uploadMutation.isPending ? 'Uploading...' : 'Upload & Parse'}
              </button>
            </div>
          )}
        </>
      )}

      {step === 'preview' && preview && (
        <div className={styles.previewSection}>
          <h2>Preview</h2>
          <div className={styles.stats}>
            <div className={styles.stat}>
              <span className={styles.statValue}>{preview.bills.length}</span>
              <span className={styles.statLabel}>Bills found</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statValue}>{preview.period_count}</span>
              <span className={styles.statLabel}>Pay periods</span>
            </div>
          </div>

          {preview.warnings.length > 0 && (
            <div className={styles.warnings}>
              <AlertTriangle size={16} />
              <div>
                {preview.warnings.map((w, i) => <p key={i}>{w}</p>)}
              </div>
            </div>
          )}

          <div className={styles.billTable}>
            <div className={styles.tableHeader}>
              <span>Name</span>
              <span>Due Day</span>
              <span>Amount</span>
              <span>Auto</span>
              <span>Category</span>
            </div>
            {preview.bills.map((bill, i) => (
              <div key={i} className={styles.tableRow}>
                <span className={styles.billName}>{bill.name}</span>
                <span>{bill.due_day || '-'}</span>
                <span>{bill.default_amount ? `$${bill.default_amount}` : '-'}</span>
                <span>{bill.is_autopay ? 'Yes' : 'No'}</span>
                <span>{bill.category || '-'}</span>
              </div>
            ))}
          </div>

          <div className={styles.previewActions}>
            <button className={styles.cancelBtn} onClick={() => { setStep('upload'); setPreview(null); }}>
              Back
            </button>
            <button className={styles.primaryBtn} onClick={() => confirmMutation.mutate()} disabled={confirmMutation.isPending}>
              {confirmMutation.isPending ? 'Importing...' : 'Confirm Import'}
            </button>
          </div>
        </div>
      )}

      {error && <div className={styles.error}>{error}</div>}
    </div>
  );
}
