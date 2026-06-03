import { type ReactNode, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { clearRouteChunkReloadAttempt } from '@/utils/routeChunkRecovery';

interface RouteLoadSuccessMarkerProps {
  children: ReactNode;
}

export const RouteLoadSuccessMarker = ({ children }: RouteLoadSuccessMarkerProps) => {
  const location = useLocation();

  useEffect(() => {
    if (typeof window === 'undefined') return;

    clearRouteChunkReloadAttempt(location.pathname, window.sessionStorage);
  }, [location.pathname]);

  return <>{children}</>;
};
