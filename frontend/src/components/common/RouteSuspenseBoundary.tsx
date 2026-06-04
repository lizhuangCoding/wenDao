import { type ReactNode, Suspense } from 'react';
import { useLocation } from 'react-router-dom';
import { Loading } from './Loading';
import { RouteLoadSuccessMarker } from './RouteLoadSuccessMarker';
import { RouteTransition } from './RouteTransition';

interface RouteSuspenseBoundaryProps {
  children: ReactNode;
}

export const RouteSuspenseBoundary = ({ children }: RouteSuspenseBoundaryProps) => {
  const location = useLocation();

  return (
    <Suspense key={location.pathname} fallback={<Loading />}>
      <RouteLoadSuccessMarker>
        <RouteTransition>{children}</RouteTransition>
      </RouteLoadSuccessMarker>
    </Suspense>
  );
};
