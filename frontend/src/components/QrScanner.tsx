// Station QR scanner: BarcodeDetector (native where available, the
// barcode-detector ponyfill elsewhere) over a getUserMedia stream, with a
// manual hash entry fallback for camera-denied devices and development.
// The decoded text is relayed verbatim; the server alone maps hashes to
// stations (they are never sent to clients).

import { useEffect, useRef, useState } from 'react';
import { BarcodeDetector } from 'barcode-detector/pure';

interface Props {
  onScan: (text: string) => void;
}

export function QrScanner({ onScan }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [manual, setManual] = useState('');
  const [scanning, setScanning] = useState(false);

  useEffect(() => {
    if (!scanning) return;
    let stream: MediaStream | null = null;
    let timer: number | null = null;
    let cancelled = false;

    const detector = new BarcodeDetector({ formats: ['qr_code'] });

    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: 'environment' } })
      .then((s) => {
        if (cancelled) {
          s.getTracks().forEach((t) => t.stop());
          return;
        }
        stream = s;
        const video = videoRef.current;
        if (!video) return;
        video.srcObject = s;
        void video.play();

        timer = window.setInterval(async () => {
          if (!videoRef.current || videoRef.current.readyState < 2) return;
          try {
            const codes = await detector.detect(videoRef.current);
            if (codes.length > 0 && codes[0].rawValue) {
              onScan(codes[0].rawValue);
            }
          } catch {
            /* transient decode errors are expected */
          }
        }, 400);
      })
      .catch((err: unknown) => {
        setCameraError(err instanceof Error ? err.message : 'camera unavailable');
      });

    return () => {
      cancelled = true;
      if (timer !== null) clearInterval(timer);
      stream?.getTracks().forEach((t) => t.stop());
    };
  }, [scanning, onScan]);

  return (
    <div className="qr-scanner">
      {!scanning ? (
        <button onClick={() => setScanning(true)}>Scan station QR</button>
      ) : (
        <>
          <video ref={videoRef} muted playsInline />
          {cameraError && <p className="muted">Camera error: {cameraError}</p>}
          <button onClick={() => setScanning(false)}>Stop scanning</button>
        </>
      )}
      <details>
        <summary>Enter station code manually</summary>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (manual.trim()) {
              onScan(manual.trim());
              setManual('');
            }
          }}
        >
          <input
            value={manual}
            onChange={(e) => setManual(e.target.value)}
            placeholder="station code"
            maxLength={128}
          />
          <button type="submit">Check in</button>
        </form>
      </details>
    </div>
  );
}
