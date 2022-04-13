#!/usr/bin/env python3
import sys
import math as math
import numpy as np

if not (len(sys.argv) == 2):
    print("not Enough argument")
    print("python calcDebitG5k.py [nb_node] ")
    exit(1)

nb_val_list = [4, 50, 75, 100, 125, 150, 175, 200]
tx_est = [150.2, 50., 35., 25.65, 20.37, 17.07, 14.65, 12.82]

nb_val = int(sys.argv[1])
securite = 12

if __name__ == '__main__':
    # a = np.concatenate((np.arange(4, 19),
    #                     np.arange(19, 47, 2),
    #                     np.arange(50, 100, 5),
    #                     np.arange(100, 201, 10),
    #                     ))
    # print(len(a))

    i = 0
    while not (nb_val_list[i] <= nb_val <= nb_val_list[i + 1]):
        i += 1
    alpha = (tx_est[i + 1] - tx_est[i]) / (nb_val_list[i + 1] - nb_val_list[i])
    beta = tx_est[i] - alpha * nb_val_list[i]
    print(int(alpha * nb_val + beta) + securite)
