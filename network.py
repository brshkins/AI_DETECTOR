import torch
import torch.nn as nn

# --- Depthwise-Separable блок ---
class DSConv(nn.Module):
    def __init__(self, in_ch, out_ch, k=3, s=1):
        super().__init__()
        p = k // 2
        self.block = nn.Sequential(
            nn.Conv2d(in_ch, in_ch, k, s, p, groups=in_ch, bias=False),  # depthwise
            nn.BatchNorm2d(in_ch),
            nn.ReLU(inplace=True),
            nn.Conv2d(in_ch, out_ch, 1, 1, 0, bias=False),               # pointwise
            nn.BatchNorm2d(out_ch),
            nn.ReLU(inplace=True),
        )
    def forward(self, x): return self.block(x)

# --- Лёгкий CNN-бэкбон для кропов глаз/рта ---
class TinyBackbone(nn.Module):
    def __init__(self, out_dim=64, in_ch=3):
        super().__init__()
        self.stem = nn.Sequential(
            nn.Conv2d(in_ch, 16, 3, 2, 1, bias=False),  # 112->56
            nn.BatchNorm2d(16), nn.ReLU(inplace=True),
        )
        self.stage = nn.Sequential(
            DSConv(16, 32, s=2),   # 56->28
            DSConv(32, 64, s=2),   # 28->14
            DSConv(64, 96, s=2),   # 14->7
        )
        self.head = nn.Sequential(
            nn.AdaptiveAvgPool2d(1),
            nn.Flatten(),
            nn.Linear(96, out_dim)
        )
    def forward(self, x):
        x = self.stem(x)
        x = self.stage(x)
        return self.head(x)   # [B, out_dim]

class DrowsinessNet(nn.Module):
    def __init__(self, eye_dim=64, mouth_dim=64, pose_dim=3, hidden=128, num_layers=1):
        super().__init__()
        self.eye_net = TinyBackbone(out_dim=eye_dim)
        self.mouth_net = TinyBackbone(out_dim=mouth_dim)
        feat_dim = eye_dim + mouth_dim + pose_dim  # 64+64+3 = 131

        self.lstm = nn.LSTM(
            input_size=feat_dim, hidden_size=hidden,
            num_layers=num_layers, batch_first=True, bidirectional=True
        )
        self.fc = nn.Sequential(
            nn.Linear(hidden*2, 128),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3),
            nn.Linear(128, 1),
            nn.Sigmoid()
        )

    def forward(self, eyes, mouths, poses):
        # eyes, mouths: [B,T,3,H,W], poses: [B,T,3]
        B, T = eyes.size(0), eyes.size(1)

        eyes_flat   = eyes.view(B*T, 3, eyes.size(-2), eyes.size(-1))
        mouths_flat = mouths.view(B*T, 3, mouths.size(-2), mouths.size(-1))

        f_eye   = self.eye_net(eyes_flat)     # [B*T, eye_dim]
        f_mouth = self.mouth_net(mouths_flat) # [B*T, mouth_dim]
        f_pose  = poses.view(B*T, -1)         # [B*T, 3]

        feats = torch.cat([f_eye, f_mouth, f_pose], dim=1)  # [B*T, 131]
        feats = feats.view(B, T, -1)                        # [B, T, 131]

        h, _ = self.lstm(feats)             # [B, T, 2*hidden]
        h = h.mean(1)                       # [B, 2*hidden]
        return self.fc(h).squeeze(1)        # [B]