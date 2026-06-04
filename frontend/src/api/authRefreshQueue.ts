interface PendingAuthRetry {
  resolve: () => void;
  reject: (error: unknown) => void;
}

export const createAuthRefreshQueue = () => {
  let pendingRequests: PendingAuthRetry[] = [];

  const drain = () => {
    const requests = pendingRequests;
    pendingRequests = [];
    return requests;
  };

  return {
    add: (resolve: () => void, reject: (error: unknown) => void) => {
      pendingRequests.push({ resolve, reject });
    },

    resolveAll: () => {
      drain().forEach(({ resolve }) => resolve());
    },

    rejectAll: (error: unknown) => {
      drain().forEach(({ reject }) => reject(error));
    },

    size: () => pendingRequests.length,
  };
};
