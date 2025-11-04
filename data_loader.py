import torch
from torch.utils.data import DataLoader, random_split
from torchvision import datasets, transforms as T
import os
from pathlib import Path

def get_loaders(
    data_dir,
    img_size=224,
    batch_size=32,
    num_workers=0,
    train_ratio=0.8,
    val_ratio=0.1,
    seed=42
):
    g = torch.Generator().manual_seed(seed)

    train_tfms = T.Compose([
        T.Resize((img_size, img_size)),
        T.RandomHorizontalFlip(),
        T.ColorJitter(0.2, 0.2, 0.2, 0.1),
        T.RandomAffine(degrees=10, translate=(0.02,0.02), scale=(0.95,1.05)),
        T.ToTensor(),
        T.Normalize([0.5,0.5,0.5],[0.5,0.5,0.5])
    ])

    test_tfms = T.Compose([
        T.Resize((img_size, img_size)),
        T.ToTensor(),
        T.Normalize([0.5,0.5,0.5],[0.5,0.5,0.5])
    ])

    ds = datasets.ImageFolder(data_dir, transform=train_tfms)
    classes = ds.classes

    n = len(ds)
    n_train = int(n * train_ratio)
    n_val = int(n * val_ratio)
    n_test = n - n_train - n_val
    train_ds, val_ds, test_ds = random_split(ds, [n_train, n_val, n_test], generator=g)

    # у вал/теста меняем трансформации
    val_ds.dataset = datasets.ImageFolder(data_dir, transform=test_tfms)
    test_ds.dataset = val_ds.dataset

    train_loader = DataLoader(train_ds, batch_size=batch_size, shuffle=True,  num_workers=num_workers)
    val_loader = DataLoader(val_ds,   batch_size=batch_size, shuffle=False, num_workers=num_workers)
    test_loader = DataLoader(test_ds,  batch_size=batch_size, shuffle=False, num_workers=num_workers)

    return train_loader, val_loader, test_loader, classes