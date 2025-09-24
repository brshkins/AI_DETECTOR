import torch
from torch.utils.data import DataLoader, ConcatDataset, random_split
import torchvision.transforms as T
from new_dataset import SequenceDrowsinessDataset


def get_sequence_loaders(
    root_dir, batch_size=8, img_size=224, num_workers=2,
    train_ratio=0.7, val_ratio=0.15, test_ratio=0.15
):
    # трансформации
    transform = T.Compose([
        T.Resize((img_size, img_size)),
        T.ToTensor(),
        T.Normalize(mean=[0.5, 0.5, 0.5], std=[0.5, 0.5, 0.5])
    ])

    # загрузка классов
    drowsy_ds = SequenceDrowsinessDataset(root_dir, split="Drowsy", transform=transform)
    non_drowsy_ds = SequenceDrowsinessDataset(root_dir, split="Non Drowsy", transform=transform)

    full_dataset = ConcatDataset([drowsy_ds, non_drowsy_ds])

    # размеры выборок
    total = len(full_dataset)
    train_size = int(train_ratio * total)
    val_size = int(val_ratio * total)
    test_size = total - train_size - val_size

    train_ds, val_ds, test_ds = random_split(full_dataset, [train_size, val_size, test_size])

    # DataLoader'ы
    train_loader = DataLoader(train_ds, batch_size=batch_size, shuffle=True, num_workers=num_workers)
    val_loader = DataLoader(val_ds, batch_size=batch_size, shuffle=False, num_workers=num_workers)
    test_loader = DataLoader(test_ds, batch_size=batch_size, shuffle=False, num_workers=num_workers)

    return train_loader, val_loader, test_loader, ["Drowsy", "Non Drowsy"]