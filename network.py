import torch
import torch.nn as nn
import torchvision.models as models


# CNN для обработки изображений (глаза + рот)
class Face(nn.Module):
    def __init__(self, out_dim=128, weights=True):
        super().__init__()
        base = models.resnet18(
            weights=models.ResNet18_Weights.DEFAULT if weights else None
        )
        base.fc = nn.Linear(base.fc.in_features, out_dim)  # -> [B, out_dim]
        self.cnn = base

    def forward(self, x):
        return self.cnn(x)


#основная модель
class DrowsinessNet(nn.Module):
    def __init__(self, feat_dim=259, hidden=256, num_layers=2):
        super().__init__()

        self.eye_net = Face(128)
        self.mouth_net = Face(128)

        # двунаправленный LSTM для обработки временных последовательностей
        self.lstm = nn.LSTM(
            input_size=feat_dim,
            hidden_size=hidden,
            num_layers=num_layers,
            batch_first=True,
            bidirectional=True
        )

        # голова классификатора
        self.fc = nn.Sequential(
            nn.Linear(hidden * 2, 256),
            nn.ReLU(),
            nn.Dropout(0.3),
            nn.Linear(256, 1),
            nn.Sigmoid()
        )

    def forward(self, eyes, mouths, poses):
        B, T = eyes.size(0), eyes.size(1)

        # изменение формы входных данных (объединяет батч и время)
        eyes = eyes.view(B * T, 3, eyes.size(-2), eyes.size(-1))
        mouths = mouths.view(B * T, 3, mouths.size(-2), mouths.size(-1))

        # извлечение признаков
        f_eye = self.eye_net(eyes)
        f_mouth = self.mouth_net(mouths)
        f_pose = poses.view(B * T, -1)

        # обработка временных зависимостей
        feats = torch.cat([f_eye, f_mouth, f_pose], dim=1)
        feats = feats.view(B, T, -1)

        # LSTM по времени
        h, _ = self.lstm(feats)
        h = h.mean(dim=1)

        prob = self.fc(h).squeeze(1)
        return prob