import type { PaginationParams } from '@/types';

export const toPaginationQuery = <T extends PaginationParams>(params: T) => {
  const { pageSize, ...rest } = params;
  return {
    ...rest,
    page_size: pageSize,
  };
};
