import os
import torch
from torch.utils.data import Dataset
from PIL import Image
import pandas as pd
import torchvision.transforms as T


class SequenceDrowsinessDataset(Dataset):
    def __init__(self, root_dir, split="Drowsy", transform=None):
        self.root_dir = os.path.join(root_dir, split)
        self.transform = transform
        self.label = 0 if split == "Drowsy" else 1

        # загружаем CSV с позами
        self.pose_df = pd.read_csv(os.path.join(self.root_dir, "poses.csv"))

        # список файлов
        self.samples = []
        for idx, row in self.pose_df.iterrows():
            fname = row["file"]
            eye_path = os.path.join(self.root_dir, "eyes", fname)
            mouth_path = os.path.join(self.root_dir, "mouths", fname)
            if os.path.exists(eye_path) and os.path.exists(mouth_path):
                self.samples.append({
                    "eye": eye_path,
                    "mouth": mouth_path,
                    "pose": [row["pitch"], row["yaw"], row["roll"]],
                    "label": self.label
                })

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        sample = self.samples[idx]

        eye_img = Image.open(sample["eye"]).convert("RGB")
        mouth_img = Image.open(sample["mouth"]).convert("RGB")

        if self.transform:
            eye_img = self.transform(eye_img)
            mouth_img = self.transform(mouth_img)

        pose = torch.tensor(sample["pose"], dtype=torch.float32)  # [3]
        label = torch.tensor(sample["label"], dtype=torch.float32)

        # возвращаем один кадр (T=1) для совместимости с LSTM
        eyes = eye_img.unsqueeze(0)     # [T=1, 3, H, W]
        mouths = mouth_img.unsqueeze(0) # [T=1, 3, H, W]
        poses = pose.unsqueeze(0)       # [T=1, 3]

        return eyes, mouths, poses, label