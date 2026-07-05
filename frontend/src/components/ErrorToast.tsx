import { useEffect } from 'react';
import type { ErrorPayload } from '../protocol/events';

interface Props {
  error: ErrorPayload | null;
  onDismiss: () => void;
}

export function ErrorToast({ error, onDismiss }: Props) {
  useEffect(() => {
    if (!error) return;
    const timer = setTimeout(onDismiss, 6000);
    return () => clearTimeout(timer);
  }, [error, onDismiss]);

  if (!error) return null;
  return (
    <div className="error-toast" onClick={onDismiss}>
      <strong>{error.errorCode}</strong>
      <p>{error.details || error.message}</p>
    </div>
  );
}
