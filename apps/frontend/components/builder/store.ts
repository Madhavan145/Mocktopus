import { create } from 'zustand';

type BuilderUIState = {
  selectedNodeId: string | null;
  setSelectedNodeId: (id: string | null) => void;
};

export const useBuilderUIStore = create<BuilderUIState>((set) => ({
  selectedNodeId: null,
  setSelectedNodeId: (id) => set({ selectedNodeId: id }),
}));
