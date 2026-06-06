import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
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
      showToast('角色已更新', 'success');
    },
    onError: (error: any) => showToast(error.message || '更新失败', 'error'),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => userApi.updateUserStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      showToast('状态已更新', 'success');
      setBanConfirm(null);
    },
    onError: (error: any) => showToast(error.message || '更新失败', 'error'),
  });

  const roleLabel = { admin: '管理员', user: '用户' } as Record<string, string>;
  const statusLabel = { active: '正常', banned: '已封禁' } as Record<string, string>;

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
      <PageHeader title="用户管理" />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
          <TextInput
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            placeholder="搜索用户名或邮箱"
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={roleFilter}
            onChange={(event) => {
              setRoleFilter(event.target.value);
              setPage(1);
            }}
          >
            <option value="">全部角色</option>
            <option value="admin">管理员</option>
            <option value="user">用户</option>
          </SelectInput>
          <SelectInput
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="banned">已封禁</option>
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">搜索</Button>
            <Button variant="secondary" onClick={resetFilters}>重置</Button>
          </div>
        </form>
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || '用户列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            users.length === 0 ? (
              <EmptyState title="暂无用户" description="当前筛选条件下没有用户。" className="m-6" />
            ) : null
          }
        >
          <thead>
            <DataTableHeadRow>
              <DataTableHeaderCell>用户</DataTableHeaderCell>
              <DataTableHeaderCell>邮箱</DataTableHeaderCell>
              <DataTableHeaderCell>角色</DataTableHeaderCell>
              <DataTableHeaderCell>状态</DataTableHeaderCell>
              <DataTableHeaderCell>注册时间</DataTableHeaderCell>
              <DataTableHeaderCell align="right">操作</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {users.map((user) => (
              <DataTableRow key={user.id}>
                <DataTableCell>
                  <div className="flex items-center gap-3">
                    <img
                      src={user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.username}`}
                      alt={user.username}
                      className="w-8 h-8 rounded-full shrink-0"
                    />
                    <span className="font-medium text-neutral-800 dark:text-neutral-200 truncate">
                      {user.username}
                    </span>
                  </div>
                </DataTableCell>
                <DataTableCell className="text-neutral-600 dark:text-neutral-400 max-w-[200px] truncate">
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
                      {user.role === 'admin' ? '取消管理' : '设为管理员'}
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
                      {user.status === 'active' ? '封禁' : '解封'}
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
        title={banConfirm?.action === 'ban' ? '封禁用户' : '解封用户'}
        message={`确定要${banConfirm?.action === 'ban' ? '封禁' : '解封'}该用户吗？`}
        confirmText={banConfirm?.action === 'ban' ? '封禁' : '解封'}
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
