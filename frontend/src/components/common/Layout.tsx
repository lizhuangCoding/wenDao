import { ReactNode } from 'react';
import { Header } from './Header';
import { Footer } from './Footer';

interface LayoutProps {
  children: ReactNode;
  hideHeader?: boolean;
  hideFooter?: boolean;
}

export const Layout = ({ children, hideHeader = false, hideFooter = false }: LayoutProps) => {
  return (
    <div className="min-h-screen flex flex-col bg-white dark:bg-neutral-900">
      {!hideHeader && <Header />}
      <main className={`flex-1 ${hideHeader ? 'pt-0' : 'pt-16'}`}>{children}</main>
      {!hideFooter && <Footer />}
    </div>
  );
};
