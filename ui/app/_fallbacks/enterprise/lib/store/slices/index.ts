import { createSlice, PayloadAction } from "@reduxjs/toolkit";

interface ScimState {
  activeTab: "users" | "groups" | "config";
  provider: string | null;
}

const scimInitialState: ScimState = {
  activeTab: "users",
  provider: null,
};

export const scimSlice = createSlice({
  name: "enterprise/scim",
  initialState: scimInitialState,
  reducers: {
    setScimTab(state, action: PayloadAction<ScimState["activeTab"]>) {
      state.activeTab = action.payload;
    },
    setScimProvider(state, action: PayloadAction<string | null>) {
      state.provider = action.payload;
    },
  },
});

export const { setScimTab, setScimProvider } = scimSlice.actions;
export const scimReducer = scimSlice.reducer;

interface UserGovernanceState {
  selectedUserId: string | null;
  selectedTeamId: string | null;
  selectedBusinessUnitId: string | null;
}

const userGovernanceInitialState: UserGovernanceState = {
  selectedUserId: null,
  selectedTeamId: null,
  selectedBusinessUnitId: null,
};

export const userGovernanceSlice = createSlice({
  name: "enterprise/userGovernance",
  initialState: userGovernanceInitialState,
  reducers: {
    selectUser(state, action: PayloadAction<string | null>) {
      state.selectedUserId = action.payload;
    },
    selectTeam(state, action: PayloadAction<string | null>) {
      state.selectedTeamId = action.payload;
    },
    selectBusinessUnit(state, action: PayloadAction<string | null>) {
      state.selectedBusinessUnitId = action.payload;
    },
  },
});

export const { selectUser, selectTeam, selectBusinessUnit } = userGovernanceSlice.actions;
export const userReducer = userGovernanceSlice.reducer;

interface GuardrailState {
  activeRuleId: string | null;
  activeProviderId: string | null;
  filterMode: "all" | "enabled" | "disabled";
}

const guardrailInitialState: GuardrailState = {
  activeRuleId: null,
  activeProviderId: null,
  filterMode: "all",
};

export const guardrailSlice = createSlice({
  name: "enterprise/guardrails",
  initialState: guardrailInitialState,
  reducers: {
    setActiveRule(state, action: PayloadAction<string | null>) {
      state.activeRuleId = action.payload;
    },
    setActiveProvider(state, action: PayloadAction<string | null>) {
      state.activeProviderId = action.payload;
    },
    setFilterMode(state, action: PayloadAction<GuardrailState["filterMode"]>) {
      state.filterMode = action.payload;
    },
  },
});

export const { setActiveRule, setActiveProvider, setFilterMode } = guardrailSlice.actions;
export const guardrailReducer = guardrailSlice.reducer;

interface EvalsState {
  selectedDatasetId: string | null;
  comparingRunIds: string[];
}

const evalsInitialState: EvalsState = {
  selectedDatasetId: null,
  comparingRunIds: [],
};

export const evalsSlice = createSlice({
  name: "enterprise/evals",
  initialState: evalsInitialState,
  reducers: {
    selectDataset(state, action: PayloadAction<string | null>) {
      state.selectedDatasetId = action.payload;
    },
    toggleCompareRun(state, action: PayloadAction<string>) {
      const idx = state.comparingRunIds.indexOf(action.payload);
      if (idx === -1) {
        state.comparingRunIds.push(action.payload);
      } else {
        state.comparingRunIds.splice(idx, 1);
      }
    },
    clearCompareRuns(state) {
      state.comparingRunIds = [];
    },
  },
});

export const { selectDataset, toggleCompareRun, clearCompareRuns } = evalsSlice.actions;

interface AccessProfilesState {
  selectedProfileId: string | null;
}

const accessProfilesInitialState: AccessProfilesState = {
  selectedProfileId: null,
};

export const accessProfilesSlice = createSlice({
  name: "enterprise/accessProfiles",
  initialState: accessProfilesInitialState,
  reducers: {
    selectProfile(state, action: PayloadAction<string | null>) {
      state.selectedProfileId = action.payload;
    },
  },
});

export const { selectProfile } = accessProfilesSlice.actions;

interface ScopedKeysState {
  selectedKeyId: string | null;
}

const scopedKeysInitialState: ScopedKeysState = {
  selectedKeyId: null,
};

export const scopedKeysSlice = createSlice({
  name: "enterprise/scopedKeys",
  initialState: scopedKeysInitialState,
  reducers: {
    selectKey(state, action: PayloadAction<string | null>) {
      state.selectedKeyId = action.payload;
    },
  },
});

export const { selectKey } = scopedKeysSlice.actions;

interface PromptDeployState {
  selectedDeploymentId: string | null;
  promoteCanaryVersion: string | null;
}

const promptDeployInitialState: PromptDeployState = {
  selectedDeploymentId: null,
  promoteCanaryVersion: null,
};

export const promptDeploySlice = createSlice({
  name: "enterprise/promptDeploy",
  initialState: promptDeployInitialState,
  reducers: {
    selectDeployment(state, action: PayloadAction<string | null>) {
      state.selectedDeploymentId = action.payload;
    },
    setPromoteCanary(state, action: PayloadAction<string | null>) {
      state.promoteCanaryVersion = action.payload;
    },
  },
});

export const { selectDeployment, setPromoteCanary } = promptDeploySlice.actions;

export const reducers = {
  scim: scimReducer,
  userGovernance: userReducer,
  guardrails: guardrailReducer,
  evals: evalsSlice.reducer,
  accessProfiles: accessProfilesSlice.reducer,
  scopedKeys: scopedKeysSlice.reducer,
  promptDeploy: promptDeploySlice.reducer,
};

export type EnterpriseState = {
  scim: ScimState;
  userGovernance: UserGovernanceState;
  guardrails: GuardrailState;
  evals: EvalsState;
  accessProfiles: AccessProfilesState;
  scopedKeys: ScopedKeysState;
  promptDeploy: PromptDeployState;
};
