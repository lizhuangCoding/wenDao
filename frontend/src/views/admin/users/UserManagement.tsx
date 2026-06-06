import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Search } from 'lucide-react';
import { userApi } from '@/api/user';
import {
  Button,
  BulkActionBar,
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
} from '@/components/common';
import { useUIStore } from '@/store';

export const UserManagement = () => {
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [banConfirm, setBanConfirm] = useState<{ id: number; action: 'ban' | 'unban' } | null>(null);
  const [roleConfirm, setRoleConfirm] = useState<{ id: number; role: string } | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['admin-users', page, pageSize, search, roleFilter, statusFilter],
    queryFn: () =>
      userApi.listUsers({ page, pageSize, search, role: roleFilter || undefined, status: statusFilter || undefined }),
  });

  const users = data?.data ?? [];
  const totalPages = Math.max(1, data?.totalPages ?? 1);
  const currentPageIds = users.map((u) => u.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: number; role: string }) => userApi.updateUserRole(id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      showToast('角色已更新', 'success');
      setRoleConfirm(null);
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
    setSelectedIds([]);
  };

  const resetFilters = () => {
    setRoleFilter('');
    setStatusFilter('');
    setSearch('');
    setSearchInput('');
    setPage(1);
    setSelectedIds([]);
  };

  const toggleUserSelection = (id: number) => {
    setSelectedIds((ids) => (ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id]));
  };

  const toggleCurrentPageSelection = () => {
    setSelectedIds((ids) => {
      if (allCurrentPageSelected) {
        return ids.filter((id) => !currentPageIds.includes(id));
      }
      return Array.from(new Set([...ids, ...currentPageIds]));
    });
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
              setSelectedIds([]);
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
              setSelectedIds([]);
            }}
          >
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="banned">已封禁</option>
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">
              搜索
            </Button>
            <Button variant="secondary" onClick={resetFilters}>
              重置
            </Button>
          </div>
        </form>
        <BulkActionBar
          selectedCount={selectedIds.length}
          onDelete={undefined}
          onClear={() => setSelectedIds([])}
          deleteLabel=""
        />
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
              <DataTableHeaderCell>
                <input
                  type="checkbox"
                  checked={allCurrentPageSelected}
                  onChange={toggleCurrentPageSelection}
                  className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                  aria-label="选择当前页用户"
                />
              </DataTableHeaderCell>
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
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(user.id)}
                    onChange={() => toggleUserSelection(user.id)}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                    aria-label={`选择用户 ${user.username}`}
                  />
                </DataTableCell>
                <DataTableCell>
                  <div className="flex items-center gap-3">
                    <img
                      src={user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.username}`}
                      alt={user.username}
                      className="w-8 h-8 rounded-full"
                    />
                    <span className="font-medium text-neutral-800 dark:text-neutral-200">{user.username}</span>
                  </div>
                </DataTableCell>
                <DataTableCell className="text-neutral-600 dark:text-neutral-400">{user.email}</DataTableCell>
                <DataTableCell>
                  <StatusBadge variant={user.role === 'admin' ? 'warning' : 'neutral'}>
                    {roleLabel[user.role] || user.role}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell>
                  <StatusBadge variant={user.status === 'active' ? 'success' : 'danger'}>
                    {statusLabel[user.status] || user.status}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell>
                  {new Date(user.created_at).toLocaleDateString()}
                </DataTableCell>
                <DataTableCell align="right">
                  <div className="flex items-center justify-end gap-2">
                    <SelectInput
                      value={user.role}
                      onChange={(event) => {
                        setRoleConfirm({ id: user.id, role: event.target.value });
                      }}
                      className="w-24 py-1 text-xs"
                    >
                      <option value="user">用户</option>
                      <option value="admin">管理员</option>
                    </SelectInput>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const newStatus = user.status === 'active' ? 'banned' : 'active';
                        const action = newStatus === 'banned' ? 'ban' : 'unban';
                        setBanConfirm({ id: user.id, action: action as 'ban' | 'unban' });
                      }}
                      className={
                        user.status === 'active'
                          ? 'text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30'
                          : 'text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/30'
                      }
                    >
                      {user.status === 'active' ? '封禁' : '解封'}
                    </Button>
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
          onChange={(nextPage) => {
            setPage(nextPage);
            setSelectedIds([]);
          }}
          onPageSizeChange={(newSize) => {
            setPageSize(newSize);
            setPage(1);
            setSelectedIds([]);
          }}
        />
      )}

      <ConfirmModal
        isOpen={roleConfirm !== null}
        title="修改角色"
        message={`确定要将该用户的角色改为 ${roleLabel[roleConfirm?.role || '']} 吗？`}
        confirmText="确认"
        onConfirm={() => {
          if (roleConfirm) {
            roleMutation.mutate(roleConfirm);
          }
        }}
        onCancel={() => setRoleConfirm(null)}
        isConfirming={roleMutation.isPending}
      />

      <ConfirmModal
        isOpen={banConfirm !== null}
        title={banConfirm?.action === 'ban' ? '封禁用户' : '解封用户'}
        message={`确定要${banConfirm?.action === 'ban' ? '封禁' : '解封'}该用户吗？`}
        confirmText={banConfirm?.action === 'ban' ? '封禁' : '取消'}
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
