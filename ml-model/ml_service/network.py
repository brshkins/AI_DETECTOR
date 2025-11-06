import torch
import torch.nn as nn


class DSConv(nn.Module):
    def __init__(self, in_ch, out_ch, k=3, s=1):
        super().__init__()
        p = k // 2
        self.block = nn.Sequential(
            nn.Conv2d(in_ch, in_ch, k, s, p, groups=in_ch, bias=False),
            nn.BatchNorm2d(in_ch),
            nn.ReLU(inplace=True),
            nn.Conv2d(in_ch, out_ch, 1, 1, 0, bias=False),
            nn.BatchNorm2d(out_ch),
            nn.ReLU(inplace=True),
        )

    def forward(self, x): return self.block(x)


class TinyCNN(nn.Module):
    def __init__(self, num_classes=1):
        super().__init__()
        self.stem = nn.Sequential(
            nn.Conv2d(3, 16, 3, 2, 1, bias=False),  # 224->112
            nn.BatchNorm2d(16), nn.ReLU(inplace=True)
        )
        self.stage = nn.Sequential(
            DSConv(16, 32, s=2),   # 112->56
            DSConv(32, 64, s=2),   # 56->28
            DSConv(64, 96, s=2),   # 28->14
            DSConv(96, 128, s=2),  # 14->7
        )
        self.head = nn.Sequential(
            nn.AdaptiveAvgPool2d(1),
            nn.Flatten(),
            nn.Linear(128, 128),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3),
            nn.Linear(128, num_classes)  # логит
        )

    def forward(self, x):
        x = self.stem(x)
        x = self.stage(x)
        x = self.head(x)
        return x.squeeze(1)  # [B] логит