#!/usr/bin/env python3
import numpy as np

exp_duration = 3600


class Parameter:

    def __init__(self, mean: int, std: int, max_delay: int, min_delay: int = 0):
        self.mean = mean
        self.std = std
        self.max_delay = max_delay
        self.min_delay = min_delay

    def make_a_flip(self) -> int:
        delay = -1
        while not (self.min_delay <= delay <= self.max_delay):
            delay = int(np.random.normal(self.mean, self.std))
        return delay


if __name__ == '__main__':
    delay_parameter = Parameter(20, 15, 50)
    time_parameter = Parameter(150, 100, 600, min_delay=20)
    list_tuple = []
    sum_time = 0
    while sum_time < exp_duration:
        a_moment = (delay_parameter.make_a_flip(), time_parameter.make_a_flip())
        if a_moment[1] + sum_time + time_parameter.min_delay > exp_duration:
            a_moment = (a_moment[0], exp_duration - sum_time)
        list_tuple.append(a_moment)
        sum_time += a_moment[1]
    string_commands = "".join(["{}:{} ".format(a_moment[0], a_moment[1]) for a_moment in list_tuple])
    print(string_commands)
