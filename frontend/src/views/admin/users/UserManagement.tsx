import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import { userApi } from '@/api/user';
import {
  Button,
  ConfirmModal,
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHeaderCell,
  DataTableHeadRow,
  DataTableRow,
  EmptyState,
  ErrorState,
  Loading,
  PageHeader,
  Pagination,
  Panel,
  SelectInput,
  StatusBadge,
  TextInput,
  getButtonClassName,
} from '@/components/common';
import { useUIStore } from '@/store';
import { formatDate } from '@/utils';

export const UserManagement = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [banConfirm, setBanConfirm] = useState<{ id: number; action: 'ban' | 'unban' } | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['admin-users', page, pageSize, search, roleFilter, statusFilter],
    queryFn: () =>
      userApi.listUsers({ page, pageSize, search, role: roleFilter || undefined, status: statusFilter || undefined }),
  });

  const users = data?.data ?? [];
  const totalPages = Math.max(1, data?.totalPages ?? 1);

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: number; role: string }) => userApi.updateUserRole(id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      showToast(t('users.roleUpdated'), 'success');
    },
    onError: (error: any) => showToast(error.message || t('users.updateFailed'), 'error'),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => userApi.updateUserStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      showToast(t('users.statusUpdated'), 'success');
      setBanConfirm(null);
    },
    onError: (error: any) => showToast(error.message || t('users.updateFailed'), 'error'),
  });

  const roleLabel = { admin: t('users.admin'), user: t('users.user') } as Record<string, string>;
  const statusLabel = { active: t('users.normal'), banned: t('users.banned') } as Record<string, string>;

  const applySearch = (event: React.FormEvent) => {
    event.preventDefault();
    setSearch(searchInput.trim());
    setPage(1);
  };

  const resetFilters = () => {
    setRoleFilter('');
    setStatusFilter('');
    setSearch('');
    setSearchInput('');
    setPage(1);
  };

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      <PageHeader title={t('users.title')} />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
          <TextInput
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            placeholder={t('users.searchPlaceholder')}
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={roleFilter}
            onChange={(event) => {
              setRoleFilter(event.target.value);
              setPage(1);
            }}
          >
            <option value="">{t('users.allRoles')}</option>
            <option value="admin">{t('users.admin')}</option>
            <option value="user">{t('users.user')}</option>
          </SelectInput>
          <SelectInput
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <option value="">{t('users.allStatus')}</option>
            <option value="active">{t('users.normal')}</option>
            <option value="banned">{t('users.banned')}</option>
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">{t('common.search')}</Button>
            <Button variant="secondary" onClick={resetFilters}>{t('common.reset')}</Button>
          </div>
        </form>
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || t('users.loadFailed')} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            users.length === 0 ? (
              <EmptyState title={t('users.noUsers')} description={t('users.noUsersDescription')} className="m-6" />
            ) : null
          }
        >
          <thead>
            <DataTableHeadRow>
              <DataTableHeaderCell width="wide">{t('users.userColumn')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('users.emailColumn')}</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">{t('users.roleColumn')}</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">{t('users.statusColumn')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('users.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell width="actions" align="right">{t('users.actions')}</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {users.map((user) => (
              <DataTableRow key={user.id}>
                <DataTableCell className="min-w-0">
                  <div className="flex min-w-0 items-center gap-3">
                    <img
                      src={user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.username}`}
                      alt={user.username}
                      className="w-8 h-8 rounded-full shrink-0"
                    />
                    <span className="min-w-0 truncate font-medium text-neutral-800 dark:text-neutral-200" title={user.username}>
                      {user.username}
                    </span>
                  </div>
                </DataTableCell>
                <DataTableCell truncate className="text-neutral-600 dark:text-neutral-400" title={user.email}>
                  {user.email}
                </DataTableCell>
                <DataTableCell className="whitespace-nowrap">
                  <StatusBadge variant={user.role === 'admin' ? 'warning' : 'neutral'}>
                    {roleLabel[user.role] || user.role}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell className="whitespace-nowrap">
                  <StatusBadge variant={user.status === 'active' ? 'success' : 'danger'}>
                    {statusLabel[user.status] || user.status}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell className="whitespace-nowrap">
                  {formatDate(user.created_at)}
                </DataTableCell>
                <DataTableCell align="right" className="whitespace-nowrap">
                  <div className="inline-flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        const newRole = user.role === 'admin' ? 'user' : 'admin';
                        roleMutation.mutate({ id: user.id, role: newRole });
                      }}
                      disabled={roleMutation.isPending}
                      className={getButtonClassName({
                        variant: 'ghost',
                        size: 'sm',
                        className: 'text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30',
                      })}
                    >
                      {user.role === 'admin' ? t('users.revokeAdmin') : t('users.grantAdmin')}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        const newStatus = user.status === 'active' ? 'banned' : 'active';
                        const action = newStatus === 'banned' ? 'ban' : 'unban';
                        setBanConfirm({ id: user.id, action: action as 'ban' | 'unban' });
                      }}
                      className={getButtonClassName({
                        variant: 'ghost',
                        size: 'sm',
                        className:
                          user.status === 'active'
                            ? 'text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30'
                            : 'text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/30',
                      })}
                    >
                      {user.status === 'active' ? t('users.ban') : t('users.unban')}
                    </button>
                  </div>
                </DataTableCell>
              </DataTableRow>
            ))}
          </DataTableBody>
        </DataTable>
      )}

      {data && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={data?.total}
          pageSize={pageSize}
          onChange={(nextPage) => setPage(nextPage)}
          onPageSizeChange={(newSize) => {
            setPageSize(newSize);
            setPage(1);
          }}
        />
      )}

      <ConfirmModal
        isOpen={banConfirm !== null}
        title={banConfirm?.action === 'ban' ? t('users.banUserTitle') : t('users.unbanUserTitle')}
        message={banConfirm?.action === 'ban' ? t('users.banUserMessage') : t('users.unbanUserMessage')}
        confirmText={banConfirm?.action === 'ban' ? t('users.banConfirm') : t('users.unbanConfirm')}
        onConfirm={() => {
          if (banConfirm) {
            const newStatus = banConfirm.action === 'ban' ? 'banned' : 'active';
            statusMutation.mutate({ id: banConfirm.id, status: newStatus });
          }
        }}
        onCancel={() => setBanConfirm(null)}
        isConfirming={statusMutation.isPending}
        isDanger={banConfirm?.action === 'ban'}
      />
    </div>
  );
};
