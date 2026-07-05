// Route split: "/" is the player app, "/host" the host interface. Two fixed
// routes need no router library.

import { PlayerApp } from './screens/player/PlayerApp';
import { HostApp } from './screens/host/HostApp';

export function App() {
  const isHost = window.location.pathname.startsWith('/host');
  return isHost ? <HostApp /> : <PlayerApp />;
}
